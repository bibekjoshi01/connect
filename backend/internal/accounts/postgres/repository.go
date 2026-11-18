// Package postgres implements account-owned storage interfaces with PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"connect.local/api/internal/accounts"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgconn"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

var _ accounts.Repository = (*Repository)(nil)

func (r *Repository) SaveCeremony(ctx context.Context, c accounts.Ceremony) error {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.ErrUnavailable
	}
	defer tx.Rollback()
	// Serializes admission across API instances, not only within this process.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(6302212002)`); err != nil {
		return accounts.ErrUnavailable
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM connect.ceremonies WHERE id IN
  (SELECT id FROM connect.ceremonies WHERE consumed OR attempts >= 3 OR expires_at <= clock_timestamp() LIMIT 1000 FOR UPDATE SKIP LOCKED)`); err != nil {
		return accounts.ErrUnavailable
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM connect.ceremonies`).Scan(&count); err != nil {
		return accounts.ErrUnavailable
	}
	if count >= 1000 {
		return accounts.ErrLimited
	}
	if c.DeviceID != "" {
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM connect.ceremonies WHERE device_id=$1 AND NOT consumed AND attempts<3 AND expires_at>clock_timestamp()`, c.DeviceID).Scan(&count); err != nil {
			return accounts.ErrUnavailable
		}
		if count >= 3 {
			return accounts.ErrLimited
		}
	}
	data, err := json.Marshal(c.Data)
	if err != nil {
		return accounts.ErrUnavailable
	}
	var username, device any
	if c.Operation == accounts.Registration {
		username = c.Username
	}
	if c.DeviceID != "" {
		device = c.DeviceID
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO connect.ceremonies(id,operation,username,user_handle,device_id,verifier_session,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, c.ID, c.Operation, username, c.UserID, device, data, c.ExpiresAt)
	if err != nil || tx.Commit() != nil {
		return accounts.ErrUnavailable
	}
	return nil
}

const deviceSelect = `SELECT d.id,a.id,a.username,a.call_id,d.user_handle,d.credential
 FROM connect.devices d JOIN connect.accounts a ON a.id=d.account_id AND a.active_device_id=d.id `

type scanner interface{ Scan(...any) error }

func scanDevice(row scanner) (accounts.Device, error) {
	var d accounts.Device
	var raw []byte
	err := row.Scan(&d.ID, &d.Profile.AccountID, &d.Profile.Username, &d.Profile.CallID, &d.UserID, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return d, accounts.ErrUnauthorized
	}
	if err != nil || json.Unmarshal(raw, &d.Credential) != nil {
		return accounts.Device{}, accounts.ErrUnavailable
	}
	return d, nil
}
func (r *Repository) FindDevice(ctx context.Context, digest []byte) (accounts.Device, error) {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	return scanDevice(r.db.QueryRowContext(ctx, deviceSelect+`WHERE d.auth_handle_digest=$1`, digest))
}
func ceremony(ctx context.Context, tx *sql.Tx, id, operation string) (accounts.Ceremony, error) {
	var c accounts.Ceremony
	var raw, pending []byte
	err := tx.QueryRowContext(ctx, `SELECT id,operation,COALESCE(username,''),user_handle,COALESCE(device_id::text,''),verifier_session,expires_at,pending_credential
 FROM connect.ceremonies WHERE id=$1 AND operation=$2 AND NOT consumed AND attempts<3 AND expires_at>clock_timestamp() FOR UPDATE`, id, operation).
		Scan(&c.ID, &c.Operation, &c.Username, &c.UserID, &c.DeviceID, &raw, &c.ExpiresAt, &pending)
	if errors.Is(err, sql.ErrNoRows) {
		return c, accounts.ErrUnauthorized
	}
	if err != nil || json.Unmarshal(raw, &c.Data) != nil {
		return accounts.Ceremony{}, accounts.ErrUnavailable
	}
	if len(pending) > 0 {
		var credential webauthn.Credential
		if json.Unmarshal(pending, &credential) != nil {
			return accounts.Ceremony{}, accounts.ErrUnavailable
		}
		c.PendingCredential = &credential
	}
	return c, nil
}

// PrepareRegistration transitions a locked ceremony without creating identity
// rows. The same slot and failure budget are retained, so admission stays bounded.
func (r *Repository) PrepareRegistration(ctx context.Context, id string, verify func(accounts.Ceremony) (accounts.PreparedRegistration, error)) (accounts.Options, error) {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	defer tx.Rollback()
	c, err := ceremony(ctx, tx, id, accounts.Registration)
	if err != nil {
		return accounts.Options{}, err
	}
	proof, err := verify(c)
	if errors.Is(err, accounts.ErrUnauthorized) {
		return accounts.Options{}, failed(ctx, tx, id)
	}
	if err != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	data, err := json.Marshal(proof.Data)
	if err != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	credential, err := json.Marshal(proof.Credential)
	if err != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE connect.ceremonies SET operation=$2,verifier_session=$3,pending_credential=$4,expires_at=$5
 WHERE id=$1 AND operation='registration' AND NOT consumed AND attempts<3 AND expires_at>clock_timestamp()`, id, accounts.RegistrationProof, data, credential, proof.Data.Expires)
	if err != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	n, err := result.RowsAffected()
	if err != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	if n != 1 {
		return accounts.Options{}, accounts.ErrUnauthorized
	}
	if tx.Commit() != nil {
		return accounts.Options{}, accounts.ErrUnavailable
	}
	return accounts.Options{CeremonyID: id, PublicKey: proof.PublicKey}, nil
}
func failed(ctx context.Context, tx *sql.Tx, id string) error {
	return failedAs(ctx, tx, id, accounts.ErrUnauthorized)
}
func failedAs(ctx context.Context, tx *sql.Tx, id string, public error) error {
	if _, err := tx.ExecContext(ctx, `UPDATE connect.ceremonies SET attempts=attempts+1 WHERE id=$1 AND attempts<3`, id); err != nil {
		return accounts.ErrUnavailable
	}
	if tx.Commit() != nil {
		return accounts.ErrUnavailable
	}
	return public
}
func consume(ctx context.Context, tx *sql.Tx, id string) error {
	result, err := tx.ExecContext(ctx, `UPDATE connect.ceremonies SET consumed=true WHERE id=$1 AND NOT consumed AND attempts<3 AND expires_at>clock_timestamp()`, id)
	if err != nil {
		return accounts.ErrUnavailable
	}
	n, err := result.RowsAffected()
	if err != nil {
		return accounts.ErrUnavailable
	}
	if n != 1 {
		return accounts.ErrUnauthorized
	}
	return nil
}
func insertSession(ctx context.Context, tx *sql.Tx, deviceID string, s accounts.Session) error {
	digest, err := accounts.Digest(s.Token)
	if err != nil {
		return accounts.ErrUnavailable
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO connect.sessions(digest,device_id,expires_at) VALUES($1,$2,$3)`, digest, deviceID, s.ExpiresAt)
	return err
}
func unique(err error) string {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return pg.ConstraintName
	}
	return ""
}
func (r *Repository) Register(ctx context.Context, id string, verify func(accounts.Ceremony) (accounts.Enrollment, error)) (accounts.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	for attempt := 0; attempt < 3; attempt++ {
		s, err, retry := r.register(ctx, id, verify)
		if !retry {
			return s, err
		}
	}
	return accounts.Session{}, accounts.ErrUnavailable
}
func (r *Repository) register(ctx context.Context, id string, verify func(accounts.Ceremony) (accounts.Enrollment, error)) (accounts.Session, error, bool) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.Session{}, accounts.ErrUnavailable, false
	}
	defer tx.Rollback()
	c, err := ceremony(ctx, tx, id, accounts.RegistrationProof)
	if err != nil {
		return accounts.Session{}, err, false
	}
	e, err := verify(c)
	if errors.Is(err, accounts.ErrUnauthorized) {
		return accounts.Session{}, failed(ctx, tx, id), false
	}
	if err != nil {
		return accounts.Session{}, accounts.ErrUnavailable, false
	}
	d := e.Device
	raw, err := json.Marshal(d.Credential)
	if err != nil {
		return accounts.Session{}, accounts.ErrUnavailable, false
	}
	if _, err = tx.ExecContext(ctx, "SAVEPOINT enrollment"); err != nil {
		return accounts.Session{}, accounts.ErrUnavailable, false
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO connect.accounts(id,username,call_id,active_device_id) VALUES($1,$2,$3,$4)`, d.Profile.AccountID, c.Username, d.Profile.CallID, d.ID)
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO connect.devices(id,account_id,user_handle,credential_id,credential,auth_handle_digest) VALUES($1,$2,$3,$4,$5,$6)`, d.ID, d.Profile.AccountID, c.UserID, d.Credential.ID, raw, e.HandleDigest)
	}
	if err == nil {
		err = insertSession(ctx, tx, d.ID, e.Session)
	}
	if err != nil {
		constraint := unique(err)
		switch constraint {
		case "accounts_pkey", "accounts_call_id_key", "devices_pkey", "devices_user_handle_key", "devices_auth_handle_digest_key", "sessions_pkey":
			return accounts.Session{}, accounts.ErrUnavailable, true
		case "accounts_username_key", "devices_credential_id_key":
			// Undo partial writes while retaining the ceremony lock and counting
			// the failed finish. Never disclose which identifier collided.
			if _, err = tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT enrollment"); err != nil {
				return accounts.Session{}, accounts.ErrUnavailable, false
			}
			return accounts.Session{}, failedAs(ctx, tx, id, accounts.ErrConflict), false
		default:
			return accounts.Session{}, accounts.ErrUnavailable, false
		}
	}
	if err = consume(ctx, tx, id); err != nil {
		return accounts.Session{}, err, false
	}
	if tx.Commit() != nil {
		return accounts.Session{}, accounts.ErrUnavailable, false
	}
	return e.Session, nil, false
}
func (r *Repository) Authenticate(ctx context.Context, id string, verify func(accounts.Ceremony, accounts.Device) (webauthn.Credential, accounts.Session, error)) (accounts.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	defer tx.Rollback()
	// Resolve the stable account ID, then lock account before ceremony or session.
	// A fresh read after that lock checks authority and the latest counter state.
	var accountID string
	err = tx.QueryRowContext(ctx, `SELECT d.account_id FROM connect.ceremonies c JOIN connect.devices d ON d.id=c.device_id WHERE c.id=$1`, id).Scan(&accountID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	if accountID != "" {
		var locked string
		if tx.QueryRowContext(ctx, `SELECT id FROM connect.accounts WHERE id=$1 FOR UPDATE`, accountID).Scan(&locked) != nil {
			return accounts.Session{}, accounts.ErrUnavailable
		}
	}
	c, err := ceremony(ctx, tx, id, accounts.Authentication)
	if err != nil {
		return accounts.Session{}, err
	}
	if c.DeviceID == "" {
		return accounts.Session{}, failed(ctx, tx, id)
	}
	d, err := scanDevice(tx.QueryRowContext(ctx, deviceSelect+`WHERE d.id=$1`, c.DeviceID))
	if errors.Is(err, accounts.ErrUnauthorized) {
		return accounts.Session{}, failed(ctx, tx, id)
	}
	if err != nil {
		return accounts.Session{}, err
	}
	credential, s, err := verify(c, d)
	if errors.Is(err, accounts.ErrUnauthorized) {
		return accounts.Session{}, failed(ctx, tx, id)
	}
	if err != nil {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	raw, err := json.Marshal(credential)
	if err != nil {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	if err = consume(ctx, tx, id); err != nil {
		return accounts.Session{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE connect.devices SET credential=$2 WHERE id=$1`, d.ID, raw); err != nil {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	if _, err = tx.ExecContext(ctx, `UPDATE connect.sessions SET revoked_at=clock_timestamp() WHERE device_id=$1 AND revoked_at IS NULL`, d.ID); err != nil {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	if insertSession(ctx, tx, d.ID, s) != nil || tx.Commit() != nil {
		return accounts.Session{}, accounts.ErrUnavailable
	}
	return s, nil
}
func (r *Repository) Profile(ctx context.Context, digest []byte) (accounts.Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	var p accounts.Profile
	err := r.db.QueryRowContext(ctx, `SELECT a.id,a.username,a.call_id FROM connect.sessions s
 JOIN connect.devices d ON d.id=s.device_id JOIN connect.accounts a ON a.id=d.account_id AND a.active_device_id=d.id
 WHERE s.digest=$1 AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp()`, digest).Scan(&p.AccountID, &p.Username, &p.CallID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, accounts.ErrUnauthorized
	}
	if err != nil {
		return p, accounts.ErrUnavailable
	}
	return p, nil
}
func (r *Repository) Logout(ctx context.Context, digest []byte) error {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return accounts.ErrUnavailable
	}
	defer tx.Rollback()
	var accountID string
	err = tx.QueryRowContext(ctx, `SELECT d.account_id FROM connect.sessions s JOIN connect.devices d ON d.id=s.device_id WHERE s.digest=$1`, digest).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return accounts.ErrUnauthorized
	}
	if err != nil {
		return accounts.ErrUnavailable
	}
	var locked string
	if tx.QueryRowContext(ctx, `SELECT id FROM connect.accounts WHERE id=$1 FOR UPDATE`, accountID).Scan(&locked) != nil {
		return accounts.ErrUnavailable
	}
	result, err := tx.ExecContext(ctx, `UPDATE connect.sessions s SET revoked_at=clock_timestamp()
 FROM connect.devices d,connect.accounts a WHERE s.digest=$1 AND s.device_id=d.id AND a.id=d.account_id
 AND a.active_device_id=d.id AND s.revoked_at IS NULL AND s.expires_at>clock_timestamp()`, digest)
	if err != nil {
		return accounts.ErrUnavailable
	}
	n, err := result.RowsAffected()
	if err != nil {
		return accounts.ErrUnavailable
	}
	if n != 1 {
		return accounts.ErrUnauthorized
	}
	if tx.Commit() != nil {
		return accounts.ErrUnavailable
	}
	return nil
}
func (r *Repository) Cleanup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, accounts.OperationTimeout)
	defer cancel()
	for _, query := range []string{
		`DELETE FROM connect.ceremonies WHERE id IN (SELECT id FROM connect.ceremonies WHERE consumed OR attempts>=3 OR expires_at<=clock_timestamp() LIMIT 1000 FOR UPDATE SKIP LOCKED)`,
		`DELETE FROM connect.sessions WHERE digest IN (SELECT digest FROM connect.sessions WHERE expires_at<=clock_timestamp() OR revoked_at IS NOT NULL LIMIT 1000 FOR UPDATE SKIP LOCKED)`,
	} {
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			return accounts.ErrUnavailable
		}
	}
	return nil
}
