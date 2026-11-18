//go:build integration

package integration

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connect.local/api/internal/accounts"
	accountspg "connect.local/api/internal/accounts/postgres"
	"connect.local/api/internal/httpserver"
	"connect.local/api/migrations"
	"github.com/fxamacker/cbor/v2"
)

const rp = "connect.test"
const origin = "https://connect.test"

// Synthetic, ephemeral standard WebAuthn authenticator. Keys are generated only
// in test memory; no fake-verifier route, captured response or private key fixture
// is exposed by the running application.
type authenticator struct {
	key *ecdsa.PrivateKey
	id  []byte
}

func newAuthenticator(t *testing.T) *authenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal("generate synthetic authenticator")
	}
	id := make([]byte, 32)
	_, _ = rand.Read(id)
	return &authenticator{key: key, id: id}
}
func b64(raw []byte) string { return base64.RawURLEncoding.EncodeToString(raw) }
func clientData(challenge, kind, where string) []byte {
	raw, _ := json.Marshal(map[string]any{"type": kind, "challenge": challenge, "origin": where, "crossOrigin": false})
	return raw
}
func authData(domain string, flags byte, counter uint32) []byte {
	hash := sha256.Sum256([]byte(domain))
	raw := append([]byte{}, hash[:]...)
	raw = append(raw, flags)
	return binary.BigEndian.AppendUint32(raw, counter)
}
func (a *authenticator) registration(t *testing.T, options accounts.Options, flags byte, where, domain string) json.RawMessage {
	t.Helper()
	payload, _ := json.Marshal(options.PublicKey)
	var opts struct {
		Challenge string `json:"challenge"`
	}
	_ = json.Unmarshal(payload, &opts)
	key, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: a.key.X.FillBytes(make([]byte, 32)), -3: a.key.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal("encode synthetic key")
	}
	data := authData(domain, flags|0x40, 0)
	data = append(data, make([]byte, 16)...)
	data = binary.BigEndian.AppendUint16(data, uint16(len(a.id)))
	data = append(data, a.id...)
	data = append(data, key...)
	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": data})
	if err != nil {
		t.Fatal("encode synthetic attestation")
	}
	raw, _ := json.Marshal(map[string]any{"id": b64(a.id), "rawId": b64(a.id), "type": "public-key", "authenticatorAttachment": "platform", "clientExtensionResults": map[string]any{}, "response": map[string]any{"clientDataJSON": b64(clientData(opts.Challenge, "webauthn.create", where)), "attestationObject": b64(attestation), "transports": []string{"internal"}}})
	return raw
}
func (a *authenticator) assertion(t *testing.T, options accounts.Options, flags byte, counter uint32, where, domain string, corrupt bool) json.RawMessage {
	t.Helper()
	payload, _ := json.Marshal(options.PublicKey)
	var opts struct {
		Challenge string `json:"challenge"`
	}
	_ = json.Unmarshal(payload, &opts)
	client := clientData(opts.Challenge, "webauthn.get", where)
	data := authData(domain, flags, counter)
	clientHash := sha256.Sum256(client)
	signed := append(append([]byte{}, data...), clientHash[:]...)
	digest := sha256.Sum256(signed)
	sig, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	if err != nil {
		t.Fatal("sign synthetic assertion")
	}
	if corrupt {
		sig[len(sig)-1] ^= 1
	}
	raw, _ := json.Marshal(map[string]any{"id": b64(a.id), "rawId": b64(a.id), "type": "public-key", "authenticatorAttachment": "platform", "clientExtensionResults": map[string]any{}, "response": map[string]any{"clientDataJSON": b64(client), "authenticatorData": b64(data), "signature": b64(sig)}})
	return raw
}

type accountAPI struct {
	service *accounts.Service
	server  *httptest.Server
	db      *sql.DB
}

func resetAccounts(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE connect.sessions,connect.ceremonies,connect.devices,connect.accounts CASCADE`); err != nil {
		t.Fatal("reset isolated account test data")
	}
}
func newAccountAPI(t *testing.T, db *sql.DB) *accountAPI {
	t.Helper()
	resetAccounts(t, db)
	service, err := accounts.New(accountspg.New(db), rp, []string{origin}, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httpserver.New("unused", httpserver.Options{RegisterRoutes: service.RegisterRoutes})
	server := httptest.NewTLSServer(srv.Handler)
	t.Cleanup(server.Close)
	return &accountAPI{service: service, server: server, db: db}
}
func (a *accountAPI) call(t *testing.T, method, path string, body any, token string, want int, target any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(method, a.server.URL+path, bytes.NewReader(raw))
	if err != nil {
		t.Fatal("construct test request")
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := a.server.Client().Do(req)
	if err != nil {
		t.Fatal("test HTTPS request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != want {
		t.Fatalf("%s %s status=%d want=%d", method, path, res.StatusCode, want)
	}
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("missing cache policy")
	}
	raw, err = io.ReadAll(res.Body)
	if err != nil {
		t.Fatal("read test response")
	}
	if target != nil && json.Unmarshal(raw, target) != nil {
		t.Fatal("invalid response JSON")
	}
}
func (a *accountAPI) begin(t *testing.T, username string) accounts.Options {
	t.Helper()
	var o accounts.Options
	a.call(t, "POST", "/v1/accounts/registration/options", map[string]string{"username": username}, "", 200, &o)
	return o
}
func (a *accountAPI) loginOptions(t *testing.T, handle string) accounts.Options {
	t.Helper()
	var o accounts.Options
	a.call(t, "POST", "/v1/authentication/options", map[string]string{"auth_handle": handle}, "", 200, &o)
	return o
}
func finishBody(o accounts.Options, credential json.RawMessage) any {
	return map[string]any{"ceremony_id": o.CeremonyID, "credential": credential}
}
func (a *accountAPI) prepare(t *testing.T, options accounts.Options, auth *authenticator) accounts.Options {
	t.Helper()
	var proof accounts.Options
	a.call(t, "POST", "/v1/accounts/registration/finish", finishBody(options, auth.registration(t, options, 5, origin, rp)), "", 202, &proof)
	return proof
}
func (a *accountAPI) enroll(t *testing.T, username string) (*authenticator, accounts.Session) {
	t.Helper()
	auth := newAuthenticator(t)
	proof := a.prepare(t, a.begin(t, username), auth)
	var session accounts.Session
	a.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, proof, 5, 0, origin, rp, false)), "", 201, &session)
	return auth, session
}
func (a *accountAPI) count(t *testing.T, query string, want int) {
	t.Helper()
	var count int
	if a.db.QueryRow(query).Scan(&count) != nil || count != want {
		t.Fatal("unexpected database cardinality")
	}
}

func testAccounts(t *testing.T, db *sql.DB) {
	t.Cleanup(func() { resetAccounts(t, db) })

	t.Run("migration revokes legacy sessions but preserves enrolled credentials", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, legacy := api.enroll(t, "legacy")
		var deviceID string
		var userHandle, credentialID, credential, handleDigest []byte
		if db.QueryRow(`SELECT id,user_handle,credential_id,credential,auth_handle_digest FROM connect.devices`).Scan(&deviceID, &userHandle, &credentialID, &credential, &handleDigest) != nil {
			t.Fatal("read synthetic legacy credential")
		}
		resetAccounts(t, db)
		provider, err := migrations.New(db)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = provider.Down(context.Background()); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			resetAccounts(t, db)
			if _, err := provider.Up(context.Background()); err != nil {
				t.Error("restore current schema")
			}
		})
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err = tx.Exec(`INSERT INTO connect.accounts(id,username,call_id,active_device_id) VALUES($1,$2,$3,$4)`, legacy.Profile.AccountID, legacy.Profile.Username, legacy.Profile.CallID, deviceID); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(`INSERT INTO connect.devices(id,account_id,user_handle,credential_id,credential,auth_handle_digest) VALUES($1,$2,$3,$4,$5,$6)`, deviceID, legacy.Profile.AccountID, userHandle, credentialID, credential, handleDigest); err != nil {
			t.Fatal(err)
		}
		digest, _ := accounts.Digest(legacy.Token)
		if _, err = tx.Exec(`INSERT INTO connect.sessions(digest,device_id,expires_at) VALUES($1,$2,$3)`, digest, deviceID, legacy.ExpiresAt); err != nil {
			t.Fatal(err)
		}
		if tx.Commit() != nil {
			t.Fatal("seed legacy identity state")
		}
		api.call(t, "GET", "/v1/me", nil, legacy.Token, 200, nil)
		if _, err = provider.Up(context.Background()); err != nil {
			t.Fatal(err)
		}
		api.call(t, "GET", "/v1/me", nil, legacy.Token, 401, nil)
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
		options := api.loginOptions(t, legacy.AuthHandle)
		var renewed accounts.Session
		api.call(t, "POST", "/v1/authentication/finish", finishBody(options, auth.assertion(t, options, 5, 1, origin, rp, false)), "", 200, &renewed)
		if _, err = provider.Up(context.Background()); err != nil {
			t.Fatal(err)
		}
		api.call(t, "GET", "/v1/me", nil, renewed.Token, 200, nil)
	})

	t.Run("public-key-only registration grants no account authority", func(t *testing.T) {
		api := newAccountAPI(t, db)
		original := newAuthenticator(t)
		publicOnly := &authenticator{key: &ecdsa.PrivateKey{PublicKey: original.key.PublicKey}, id: original.id}
		if publicOnly.key.D != nil {
			t.Fatal("fixture retained signing key")
		}
		options := api.begin(t, "public_only")
		var response map[string]json.RawMessage
		api.call(t, "POST", "/v1/accounts/registration/finish", finishBody(options, publicOnly.registration(t, options, 5, origin, rp)), "", 202, &response)
		if len(response) != 2 || response["token"] != nil || response["auth_handle"] != nil || response["profile"] != nil {
			t.Fatal("attestation granted account authority")
		}
		raw, _ := json.Marshal(response)
		var proof accounts.Options
		if json.Unmarshal(raw, &proof) != nil {
			t.Fatal("invalid proof options")
		}
		api.count(t, `SELECT count(*) FROM connect.accounts`, 0)
		api.count(t, `SELECT count(*) FROM connect.devices`, 0)
		api.count(t, `SELECT count(*) FROM connect.sessions`, 0)
		wrongKey := newAuthenticator(t)
		wrongKey.id = publicOnly.id
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, wrongKey.assertion(t, proof, 5, 0, origin, rp, false)), "", 401, nil)
		api.count(t, `SELECT count(*) FROM connect.accounts`, 0)
		// The actual key holder can finish the same pending enrollment.
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, original.assertion(t, proof, 5, 0, origin, rp, false)), "", 201, nil)
	})
	for _, tc := range []struct {
		name          string
		flags         byte
		where, domain string
		corrupt       bool
	}{
		{"wrong origin", 5, "https://evil.test", rp, false}, {"wrong RP", 5, origin, "evil.test", false},
		{"missing verification", 1, origin, rp, false}, {"missing presence", 4, origin, rp, false},
		{"backup eligible", 13, origin, rp, false}, {"invalid signature", 5, origin, rp, true},
	} {
		t.Run("reject enrollment proof "+tc.name, func(t *testing.T) {
			api := newAccountAPI(t, db)
			auth := newAuthenticator(t)
			proof := api.prepare(t, api.begin(t, "alice"), auth)
			api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, proof, tc.flags, 0, tc.where, tc.domain, tc.corrupt)), "", 401, nil)
			api.count(t, `SELECT count(*) FROM connect.accounts`, 0)
			api.count(t, `SELECT count(*) FROM connect.devices`, 0)
			api.count(t, `SELECT count(*) FROM connect.sessions`, 0)
		})
	}
	t.Run("enrollment proof cannot skip phases exceed attempts or outlive expiry", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth := newAuthenticator(t)
		options := api.begin(t, "alice")
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(options, auth.registration(t, options, 5, origin, rp)), "", 401, nil)
		// A failed registration consumes the same budget as later proof failures.
		api.call(t, "POST", "/v1/accounts/registration/finish", finishBody(options, auth.registration(t, options, 1, origin, rp)), "", 401, nil)
		proof := api.prepare(t, options, auth)
		api.call(t, "POST", "/v1/authentication/finish", finishBody(proof, auth.assertion(t, proof, 5, 0, origin, rp, false)), "", 401, nil)
		// A signature over the original registration challenge is not a proof
		// over the fresh assertion challenge, even with the correct private key.
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, options, 5, 0, origin, rp, false)), "", 401, nil)
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, proof, 5, 0, origin, rp, true)), "", 401, nil)
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, proof, 5, 0, origin, rp, false)), "", 401, nil)
		expiring := api.prepare(t, api.begin(t, "bob"), auth)
		if _, err := db.Exec(`UPDATE connect.ceremonies SET expires_at=clock_timestamp()-interval '1 second' WHERE id=$1`, expiring.CeremonyID); err != nil {
			t.Fatal(err)
		}
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(expiring, auth.assertion(t, expiring, 5, 0, origin, rp, false)), "", 401, nil)
		api.count(t, `SELECT count(*) FROM connect.accounts`, 0)
	})
	t.Run("HTTPS enrollment rotation logout and persisted state", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, initial := api.enroll(t, "Alice_7")
		if initial.Profile.Username != "alice_7" || len(initial.Profile.CallID) != 32 || len(initial.Profile.AccountID) != 36 || initial.AuthHandle == "" {
			t.Fatal("invalid enrollment response")
		}
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
		api.count(t, `SELECT count(*) FROM connect.devices`, 1)
		var profile accounts.Profile
		api.call(t, "GET", "/v1/me", nil, initial.Token, 200, &profile)
		if profile != initial.Profile {
			t.Fatal("incorrect profile")
		}
		o := api.loginOptions(t, initial.AuthHandle)
		var rotated accounts.Session
		api.call(t, "POST", "/v1/authentication/finish", finishBody(o, auth.assertion(t, o, 5, 1, origin, rp, false)), "", 200, &rotated)
		if rotated.Token == initial.Token || rotated.AuthHandle != "" || rotated.Profile != initial.Profile {
			t.Fatal("incorrect session rotation")
		}
		api.call(t, "GET", "/v1/me", nil, initial.Token, 401, nil)
		api.call(t, "GET", "/v1/me", nil, rotated.Token, 200, nil)
		api.count(t, `SELECT count(*) FROM connect.sessions WHERE revoked_at IS NULL`, 1)
		var counter int
		if db.QueryRow(`SELECT (credential->'authenticator'->>'signCount')::integer FROM connect.devices`).Scan(&counter) != nil || counter != 1 {
			t.Fatal("counter state not persisted")
		}
		api.call(t, "DELETE", "/v1/sessions/current", nil, rotated.Token, 204, nil)
		api.call(t, "GET", "/v1/me", nil, rotated.Token, 401, nil)
		// A fresh service has no authorization cache; stored credentials can log in.
		fresh, err := accounts.New(accountspg.New(db), rp, []string{origin}, nil)
		if err != nil {
			t.Fatal(err)
		}
		next, err := fresh.BeginAuthentication(context.Background(), initial.AuthHandle)
		if err != nil {
			t.Fatal(err)
		}
		session, err := fresh.FinishAuthentication(context.Background(), next.CeremonyID, auth.assertion(t, next, 5, 2, origin, rp, false))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = fresh.Profile(context.Background(), session.Token); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`UPDATE connect.sessions SET expires_at=clock_timestamp()-interval '1 second'`); err != nil {
			t.Fatal(err)
		}
		api.call(t, "GET", "/v1/me", nil, session.Token, 401, nil)
	})
	for _, tc := range []struct {
		name          string
		flags         byte
		where, domain string
	}{
		{"wrong origin", 5, "https://evil.test", rp}, {"wrong RP", 5, origin, "evil.test"}, {"no verification", 1, origin, rp},
		{"no presence", 4, origin, rp}, {"backup eligible", 13, origin, rp}, {"backed up", 29, origin, rp},
	} {
		t.Run("reject registration "+tc.name, func(t *testing.T) {
			api := newAccountAPI(t, db)
			auth := newAuthenticator(t)
			o := api.begin(t, "alice")
			api.call(t, "POST", "/v1/accounts/registration/finish", finishBody(o, auth.registration(t, o, tc.flags, tc.where, tc.domain)), "", 401, nil)
			api.count(t, `SELECT count(*) FROM connect.accounts`, 0)
			api.count(t, `SELECT attempts FROM connect.ceremonies`, 1)
		})
	}
	for _, tc := range []struct {
		name          string
		flags         byte
		counter       uint32
		where, domain string
		corrupt       bool
	}{
		{"wrong origin", 5, 1, "https://evil.test", rp, false}, {"wrong RP", 5, 1, origin, "evil.test", false},
		{"no verification", 1, 1, origin, rp, false}, {"backup eligible", 13, 1, origin, rp, false},
		{"backed up without eligibility", 21, 1, origin, rp, false}, {"invalid signature", 5, 1, origin, rp, true},
	} {
		t.Run("reject assertion "+tc.name, func(t *testing.T) {
			api := newAccountAPI(t, db)
			auth, initial := api.enroll(t, "alice")
			o := api.loginOptions(t, initial.AuthHandle)
			api.call(t, "POST", "/v1/authentication/finish", finishBody(o, auth.assertion(t, o, tc.flags, tc.counter, tc.where, tc.domain, tc.corrupt)), "", 401, nil)
			api.call(t, "GET", "/v1/me", nil, initial.Token, 200, nil)
			api.count(t, `SELECT count(*) FROM connect.sessions`, 1)
		})
	}
	t.Run("clone warning rejected and failed attempts exhausted", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, initial := api.enroll(t, "alice")
		o := api.loginOptions(t, initial.AuthHandle)
		api.call(t, "POST", "/v1/authentication/finish", finishBody(o, auth.assertion(t, o, 5, 2, origin, rp, false)), "", 200, nil)
		bad := api.loginOptions(t, initial.AuthHandle)
		for i := 0; i < 3; i++ {
			api.call(t, "POST", "/v1/authentication/finish", finishBody(bad, auth.assertion(t, bad, 5, 1, origin, rp, false)), "", 401, nil)
		}
		api.call(t, "POST", "/v1/authentication/finish", finishBody(bad, auth.assertion(t, bad, 5, 3, origin, rp, false)), "", 401, nil)
		api.count(t, `SELECT count(*) FROM connect.sessions`, 2)
		api.count(t, `SELECT max(attempts) FROM connect.ceremonies`, 3)
	})
	t.Run("replay wrong operation and expiry", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth := newAuthenticator(t)
		o := api.begin(t, "alice")
		response := auth.registration(t, o, 5, origin, rp)
		api.call(t, "POST", "/v1/authentication/finish", finishBody(o, response), "", 401, nil)
		var proof accounts.Options
		api.call(t, "POST", "/v1/accounts/registration/finish", finishBody(o, response), "", 202, &proof)
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, proof, 5, 0, origin, rp, false)), "", 201, nil)
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(proof, auth.assertion(t, proof, 5, 0, origin, rp, false)), "", 401, nil)
		api.call(t, "POST", "/v1/accounts/registration/finish", finishBody(o, response), "", 401, nil)
		pending := api.begin(t, "bob")
		if _, err := db.Exec(`UPDATE connect.ceremonies SET expires_at=clock_timestamp()-interval '1 second' WHERE id=$1`, pending.CeremonyID); err != nil {
			t.Fatal(err)
		}
		api.call(t, "POST", "/v1/accounts/registration/finish", finishBody(pending, auth.registration(t, pending, 5, origin, rp)), "", 401, nil)
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
	})
	t.Run("username and credential uniqueness", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, _ := api.enroll(t, "alice")
		sameUser := api.begin(t, "ALICE")
		other := newAuthenticator(t)
		sameUserProof := api.prepare(t, sameUser, other)
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(sameUserProof, other.assertion(t, sameUserProof, 5, 0, origin, rp, false)), "", 409, nil)
		reused := api.begin(t, "bob")
		reusedProof := api.prepare(t, reused, auth)
		api.call(t, "POST", "/v1/accounts/registration/proof", finishBody(reusedProof, auth.assertion(t, reusedProof, 5, 0, origin, rp, false)), "", 409, nil)
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
		api.count(t, `SELECT count(*) FROM connect.devices`, 1)
		api.count(t, `SELECT count(*) FROM connect.sessions`, 1)
	})
	t.Run("unknown handles do not authenticate", func(t *testing.T) {
		api := newAccountAPI(t, db)
		o := api.loginOptions(t, accounts.RandomValue())
		auth := newAuthenticator(t)
		api.call(t, "POST", "/v1/authentication/finish", finishBody(o, auth.assertion(t, o, 5, 1, origin, rp, false)), "", 401, nil)
		api.count(t, `SELECT count(*) FROM connect.sessions`, 0)
	})
	t.Run("concurrent registration transition proof consumption and username races", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth := newAuthenticator(t)
		options := api.begin(t, "alice")
		body := auth.registration(t, options, 5, origin, rp)
		type outcome struct {
			proof accounts.Options
			err   error
		}
		out := make(chan outcome, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				proof, err := api.service.FinishRegistration(context.Background(), options.CeremonyID, body)
				out <- outcome{proof, err}
			}()
		}
		wg.Wait()
		close(out)
		successes := 0
		var proof accounts.Options
		for result := range out {
			if result.err == nil {
				successes++
				proof = result.proof
			} else if !errors.Is(result.err, accounts.ErrUnauthorized) {
				t.Fatal("unexpected preparation race result")
			}
		}
		if successes != 1 {
			t.Fatal("preparation was not single use")
		}
		api.count(t, `SELECT count(*) FROM connect.accounts`, 0)
		raceProof := func(options []accounts.Options, auths []*authenticator) {
			bodies := make([]json.RawMessage, len(options))
			for i, o := range options {
				bodies[i] = auths[i].assertion(t, o, 5, 0, origin, rp, false)
			}
			results := make(chan error, len(options))
			var workers sync.WaitGroup
			for i, o := range options {
				workers.Add(1)
				go func() {
					defer workers.Done()
					_, err := api.service.ProveRegistration(context.Background(), o.CeremonyID, bodies[i])
					results <- err
				}()
			}
			workers.Wait()
			close(results)
			successes := 0
			for err := range results {
				if err == nil {
					successes++
				} else if !errors.Is(err, accounts.ErrUnauthorized) && !errors.Is(err, accounts.ErrConflict) {
					t.Fatal("unexpected proof race result")
				}
			}
			if successes != 1 {
				t.Fatal("enrollment proof race issued multiple sessions")
			}
		}
		raceProof([]accounts.Options{proof, proof}, []*authenticator{auth, auth})
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
		resetAccounts(t, db)
		oneAuth, twoAuth := newAuthenticator(t), newAuthenticator(t)
		one := api.prepare(t, api.begin(t, "bob"), oneAuth)
		two := api.prepare(t, api.begin(t, "bob"), twoAuth)
		raceProof([]accounts.Options{one, two}, []*authenticator{oneAuth, twoAuth})
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
	})
	t.Run("concurrent credential reuse cannot create partial accounts", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth := newAuthenticator(t)
		options := []accounts.Options{api.prepare(t, api.begin(t, "alice"), auth), api.prepare(t, api.begin(t, "bob"), auth)}
		bodies := []json.RawMessage{auth.assertion(t, options[0], 5, 0, origin, rp, false), auth.assertion(t, options[1], 5, 0, origin, rp, false)}
		var wg sync.WaitGroup
		out := make(chan error, 2)
		for i, o := range options {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := api.service.ProveRegistration(context.Background(), o.CeremonyID, bodies[i])
				out <- err
			}()
		}
		wg.Wait()
		close(out)
		successes := 0
		for err := range out {
			if err == nil {
				successes++
			} else if !errors.Is(err, accounts.ErrConflict) {
				t.Fatal("unexpected credential race result")
			}
		}
		if successes != 1 {
			t.Fatal("credential enrolled twice")
		}
		api.count(t, `SELECT count(*) FROM connect.accounts`, 1)
		api.count(t, `SELECT count(*) FROM connect.devices`, 1)
		api.count(t, `SELECT count(*) FROM connect.sessions`, 1)
	})
	t.Run("concurrent login consumption and logout", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, initial := api.enroll(t, "alice")
		o := api.loginOptions(t, initial.AuthHandle)
		body := auth.assertion(t, o, 5, 1, origin, rp, false)
		var wg sync.WaitGroup
		out := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := api.service.FinishAuthentication(context.Background(), o.CeremonyID, body)
				out <- err
			}()
		}
		wg.Wait()
		close(out)
		successes := 0
		for err := range out {
			if err == nil {
				successes++
			} else if !errors.Is(err, accounts.ErrUnauthorized) {
				t.Fatal("unexpected login race result")
			}
		}
		if successes != 1 {
			t.Fatal("login replay issued multiple sessions")
		}
		api.count(t, `SELECT count(*) FROM connect.sessions WHERE revoked_at IS NULL`, 1)
		// Logging out the replaced token cannot revoke a later session.
		api.call(t, "DELETE", "/v1/sessions/current", nil, initial.Token, 401, nil)
		api.count(t, `SELECT count(*) FROM connect.sessions WHERE revoked_at IS NULL`, 1)
	})

	t.Run("distinct ceremonies recheck the latest credential counter", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, initial := api.enroll(t, "alice")
		one := api.loginOptions(t, initial.AuthHandle)
		two := api.loginOptions(t, initial.AuthHandle)
		bodies := []json.RawMessage{auth.assertion(t, one, 5, 1, origin, rp, false), auth.assertion(t, two, 5, 1, origin, rp, false)}
		options := []accounts.Options{one, two}
		out := make(chan error, 2)
		var wg sync.WaitGroup
		for i, o := range options {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := api.service.FinishAuthentication(context.Background(), o.CeremonyID, bodies[i])
				out <- err
			}()
		}
		wg.Wait()
		close(out)
		successes := 0
		for err := range out {
			if err == nil {
				successes++
			} else if !errors.Is(err, accounts.ErrUnauthorized) {
				t.Fatal("unexpected counter race result")
			}
		}
		if successes != 1 {
			t.Fatal("stale counter accepted in concurrent login")
		}
		api.count(t, `SELECT count(*) FROM connect.sessions WHERE revoked_at IS NULL`, 1)
	})
	t.Run("inactive device loses session and login authority", func(t *testing.T) {
		api := newAccountAPI(t, db)
		auth, initial := api.enroll(t, "alice")
		o := api.loginOptions(t, initial.AuthHandle)
		// Model a future atomic authority change directly in the isolated database;
		// this does not implement or expose a device-transfer endpoint.
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		var accountID string
		if tx.QueryRow(`SELECT id FROM connect.accounts WHERE id=$1 FOR UPDATE`, initial.Profile.AccountID).Scan(&accountID) != nil {
			t.Fatal("lock authority")
		}
		_, err = tx.Exec(`INSERT INTO connect.devices(id,account_id,user_handle,credential_id,credential,auth_handle_digest)
   SELECT '00000000-0000-4000-8000-000000000001',account_id,decode(repeat('01',16),'hex'),decode(repeat('02',32),'hex'),credential,decode(repeat('03',32),'hex') FROM connect.devices WHERE account_id=$1`, accountID)
		if err != nil {
			t.Fatal("seed replacement authority")
		}
		if _, err = tx.Exec(`UPDATE connect.accounts SET active_device_id='00000000-0000-4000-8000-000000000001' WHERE id=$1`, accountID); err != nil {
			t.Fatal(err)
		}
		if tx.Commit() != nil {
			t.Fatal("commit authority change")
		}
		api.call(t, "GET", "/v1/me", nil, initial.Token, 401, nil)
		api.call(t, "DELETE", "/v1/sessions/current", nil, initial.Token, 401, nil)
		api.call(t, "POST", "/v1/authentication/finish", finishBody(o, auth.assertion(t, o, 5, 1, origin, rp, false)), "", 401, nil)
	})
	t.Run("device authority enforced and nonempty rollback refused", func(t *testing.T) {
		api := newAccountAPI(t, db)
		_, first := api.enroll(t, "alice")
		_, second := api.enroll(t, "bob")
		if _, err := db.Exec(`UPDATE connect.accounts SET active_device_id=(SELECT active_device_id FROM connect.accounts WHERE id=$1) WHERE id=$2`, second.Profile.AccountID, first.Profile.AccountID); err == nil {
			t.Fatal("cross-account active device accepted")
		}
		provider, err := migrations.New(db)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = provider.Down(context.Background()); err == nil {
			t.Fatal("nonempty identity rollback succeeded")
		}
		api.call(t, "GET", "/v1/me", nil, first.Token, 200, nil)
	})
	t.Run("database admission limits cleanup and bounded failure", func(t *testing.T) {
		api := newAccountAPI(t, db)
		_, initial := api.enroll(t, "alice")
		for i := 0; i < 3; i++ {
			if _, err := api.service.BeginAuthentication(context.Background(), initial.AuthHandle); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := api.service.BeginAuthentication(context.Background(), initial.AuthHandle); !errors.Is(err, accounts.ErrLimited) {
			t.Fatal("pending device limit ignored")
		}
		resetAccounts(t, db)
		seed, err := api.service.BeginRegistration(context.Background(), "alice")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`INSERT INTO connect.ceremonies(id,operation,username,user_handle,verifier_session,expires_at)
  SELECT substr(md5(i::text)||md5(i::text),1,43),operation,username,user_handle,verifier_session,expires_at FROM connect.ceremonies CROSS JOIN generate_series(1,998) i WHERE id=$1`, seed.CeremonyID)
		if err != nil {
			t.Fatal("seed ceremony capacity")
		}
		var wg sync.WaitGroup
		out := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := api.service.BeginRegistration(context.Background(), "bob")
				out <- err
			}()
		}
		wg.Wait()
		close(out)
		successes, limited := 0, 0
		for err := range out {
			if err == nil {
				successes++
			} else if errors.Is(err, accounts.ErrLimited) {
				limited++
			} else {
				t.Fatal(err)
			}
		}
		if successes != 1 || limited != 1 {
			t.Fatal("atomic global capacity ignored")
		}
		if _, err = db.Exec(`UPDATE connect.ceremonies SET expires_at=clock_timestamp()-interval '1 second'`); err != nil {
			t.Fatal(err)
		}
		if err = accountspg.New(db).Cleanup(context.Background()); err != nil {
			t.Fatal(err)
		}
		api.count(t, `SELECT count(*) FROM connect.ceremonies`, 0)
		blocked, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer stop()
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err = tx.Exec(`LOCK TABLE connect.sessions IN ACCESS EXCLUSIVE MODE`); err != nil {
			t.Fatal(err)
		}
		if _, err = api.service.Profile(blocked, initial.Token); !errors.Is(err, accounts.ErrUnavailable) {
			t.Fatal("blocked database did not fail closed")
		}
	})
}

func testAccountOutage(t *testing.T, db *sql.DB) {
	service, err := accounts.New(accountspg.New(db), rp, []string{origin}, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httpserver.New("unused", httpserver.Options{RegisterRoutes: service.RegisterRoutes})
	server := httptest.NewTLSServer(srv.Handler)
	defer server.Close()
	api := &accountAPI{service: service, server: server, db: db}
	api.call(t, "GET", "/v1/me", nil, accounts.RandomValue(), 503, nil)
	api.call(t, "DELETE", "/v1/sessions/current", nil, accounts.RandomValue(), 503, nil)
	api.call(t, "POST", "/v1/accounts/registration/options", map[string]string{"username": "alice"}, "", 503, nil)
	api.call(t, "POST", "/v1/authentication/options", map[string]string{"auth_handle": accounts.RandomValue()}, "", 503, nil)
	api.call(t, "POST", "/v1/accounts/registration/finish", map[string]any{"ceremony_id": accounts.RandomValue(), "credential": map[string]any{}}, "", 503, nil)
	api.call(t, "POST", "/v1/accounts/registration/proof", map[string]any{"ceremony_id": accounts.RandomValue(), "credential": map[string]any{}}, "", 503, nil)
	api.call(t, "POST", "/v1/authentication/finish", map[string]any{"ceremony_id": accounts.RandomValue(), "credential": map[string]any{}}, "", 503, nil)
}
