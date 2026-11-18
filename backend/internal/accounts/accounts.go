// Package accounts owns enrollment, device authentication and revocable sessions.
// It has no dependency on the mobile app or media engine.
package accounts

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var (
	ErrInvalid      = errors.New("invalid_request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("registration_conflict")
	ErrLimited      = errors.New("rate_limited")
	ErrUnavailable  = errors.New("unavailable")
)

const (
	Registration      = "registration"
	RegistrationProof = "registration_proof"
	Authentication    = "authentication"
	CeremonyTTL       = 2 * time.Minute
	SessionTTL        = 15 * time.Minute
	OperationTimeout  = 2 * time.Second
)

// Repository is owned by this module. Finish callbacks run under database locks
// against the current credential; no stale verification result can rotate a session.
// Unauthorized callbacks count a failed attempt, without committing other changes.
type Repository interface {
	SaveCeremony(context.Context, Ceremony) error
	FindDevice(context.Context, []byte) (Device, error)
	PrepareRegistration(context.Context, string, func(Ceremony) (PreparedRegistration, error)) (Options, error)
	Register(context.Context, string, func(Ceremony) (Enrollment, error)) (Session, error)
	Authenticate(context.Context, string, func(Ceremony, Device) (webauthn.Credential, Session, error)) (Session, error)
	Profile(context.Context, []byte) (Profile, error)
	Logout(context.Context, []byte) error
	Cleanup(context.Context) error
}

type Profile struct {
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
	CallID    string `json:"call_id"`
}
type Session struct {
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expires_at"`
	Profile    Profile   `json:"profile"`
	AuthHandle string    `json:"auth_handle,omitempty"`
}
type Ceremony struct {
	ID                string
	Operation         string
	Username          string
	UserID            []byte
	DeviceID          string
	Data              webauthn.SessionData
	ExpiresAt         time.Time
	PendingCredential *webauthn.Credential
}
type Device struct {
	ID         string
	Profile    Profile
	UserID     []byte
	Credential webauthn.Credential
}

func (d Device) WebAuthnID() []byte          { return d.UserID }
func (d Device) WebAuthnName() string        { return d.Profile.Username }
func (d Device) WebAuthnDisplayName() string { return d.Profile.Username }
func (d Device) WebAuthnCredentials() []webauthn.Credential {
	if len(d.Credential.ID) == 0 {
		return nil
	}
	return []webauthn.Credential{d.Credential}
}

type Enrollment struct {
	Device       Device
	Session      Session
	HandleDigest []byte
}
type PreparedRegistration struct {
	Credential webauthn.Credential
	Data       webauthn.SessionData
	PublicKey  any
}
type Options struct {
	CeremonyID string `json:"ceremony_id"`
	PublicKey  any    `json:"publicKey"`
}

type Service struct {
	repo     Repository
	verifier *webauthn.WebAuthn
}

// New never weakens verifier signature, origin, extension or attestation policy.
func New(repo Repository, rpID string, origins, androidOrigins []string) (*Service, error) {
	timeout := webauthn.TimeoutConfig{Enforce: true, Timeout: CeremonyTTL, TimeoutUVD: CeremonyTTL}
	verifier, err := webauthn.New(&webauthn.Config{
		RPID: rpID, RPDisplayName: "Connect", RPOrigins: origins, RPOpaqueOrigins: androidOrigins,
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementDiscouraged,
			UserVerification:        protocol.VerificationRequired,
		},
		Filtering: &webauthn.FilteringConfig{ProhibitBackupEligibility: true},
		Timeouts:  webauthn.TimeoutsConfig{Login: timeout, Registration: timeout},
	})
	if err != nil {
		return nil, errors.New("invalid WebAuthn relying-party configuration")
	}
	return &Service{repo: repo, verifier: verifier}, nil
}

var usernamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)

func Username(raw string) (string, error) {
	// ASCII folding only; Unicode lookalikes and whitespace are not normalized.
	if len(raw) < 3 || len(raw) > 32 {
		return "", ErrInvalid
	}
	for _, c := range []byte(raw) {
		if c > 127 {
			return "", ErrInvalid
		}
	}
	normalized := strings.ToLower(raw)
	if !usernamePattern.MatchString(normalized) {
		return "", ErrInvalid
	}
	return normalized, nil
}
func RandomValue() string      { return base64.RawURLEncoding.EncodeToString(randomBytes(32)) }
func randomBytes(n int) []byte { b := make([]byte, n); _, _ = rand.Read(b); return b }
func DecodeValue(raw string) ([]byte, error) {
	b, err := base64.RawURLEncoding.Strict().DecodeString(raw)
	if err != nil || len(b) != 32 || len(raw) != 43 {
		return nil, ErrInvalid
	}
	return b, nil
}
func Digest(raw string) ([]byte, error) {
	b, err := DecodeValue(raw)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	return digest[:], nil
}
func newSession(profile Profile) Session {
	return Session{Token: RandomValue(), ExpiresAt: time.Now().UTC().Add(SessionTTL), Profile: profile}
}

func (s *Service) BeginRegistration(ctx context.Context, raw string) (Options, error) {
	username, err := Username(raw)
	if err != nil {
		return Options{}, err
	}
	d := Device{UserID: randomBytes(16), Profile: Profile{Username: username}}
	creation, session, err := s.verifier.BeginRegistration(d,
		webauthn.WithCredentialParameters([]protocol.CredentialParameter{{Type: protocol.PublicKeyCredentialType, Algorithm: webauthncose.AlgES256}}))
	if err != nil {
		return Options{}, ErrUnavailable
	}
	c := Ceremony{ID: RandomValue(), Operation: Registration, Username: username, UserID: d.UserID, Data: *session, ExpiresAt: session.Expires}
	if err = s.repo.SaveCeremony(ctx, c); err != nil {
		return Options{}, err
	}
	return Options{CeremonyID: c.ID, PublicKey: creation.Response}, nil
}
func (s *Service) BeginAuthentication(ctx context.Context, handle string) (Options, error) {
	digest, err := Digest(handle)
	if err != nil {
		return Options{}, ErrInvalid
	}
	d, err := s.repo.FindDevice(ctx, digest)
	if err != nil && !errors.Is(err, ErrUnauthorized) {
		return Options{}, err
	}
	deviceID := d.ID
	if errors.Is(err, ErrUnauthorized) {
		// A dummy allowCredentials entry avoids a public credential lookup endpoint.
		d = Device{UserID: randomBytes(16), Credential: webauthn.Credential{ID: randomBytes(32)}}
	}
	assertion, session, err := s.verifier.BeginLogin(d, webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return Options{}, ErrUnavailable
	}
	c := Ceremony{ID: RandomValue(), Operation: Authentication, UserID: d.UserID, DeviceID: deviceID, Data: *session, ExpiresAt: session.Expires}
	if err = s.repo.SaveCeremony(ctx, c); err != nil {
		return Options{}, err
	}
	return Options{CeremonyID: c.ID, PublicKey: assertion.Response}, nil
}

// FinishRegistration verifies attestation but grants no account authority.
// A fresh standard assertion proves possession before final enrollment.
func (s *Service) FinishRegistration(ctx context.Context, id string, raw json.RawMessage) (Options, error) {
	if _, err := DecodeValue(id); err != nil {
		return Options{}, ErrUnauthorized
	}
	return s.repo.PrepareRegistration(ctx, id, func(c Ceremony) (PreparedRegistration, error) {
		var standard protocol.CredentialCreationResponse
		if strictCredential(raw, &standard) != nil {
			return PreparedRegistration{}, ErrUnauthorized
		}
		parsed, err := protocol.ParseCredentialCreationResponseBytes(raw)
		if err != nil {
			return PreparedRegistration{}, ErrUnauthorized
		}
		d := Device{UserID: c.UserID, Profile: Profile{Username: c.Username}}
		credential, err := s.verifier.CreateCredential(d, c.Data, parsed)
		if err != nil || credential == nil {
			return PreparedRegistration{}, ErrUnauthorized
		}
		if !acceptable(*credential) || !parsed.Response.AttestationObject.AuthData.Flags.HasUserVerified() {
			return PreparedRegistration{}, ErrUnauthorized
		}
		d.Credential = *credential
		assertion, session, err := s.verifier.BeginLogin(d, webauthn.WithUserVerification(protocol.VerificationRequired))
		if err != nil {
			return PreparedRegistration{}, ErrUnavailable
		}
		return PreparedRegistration{Credential: *credential, Data: *session, PublicKey: assertion.Response}, nil
	})
}
func (s *Service) ProveRegistration(ctx context.Context, id string, raw json.RawMessage) (Session, error) {
	if _, err := DecodeValue(id); err != nil {
		return Session{}, ErrUnauthorized
	}
	return s.repo.Register(ctx, id, func(c Ceremony) (Enrollment, error) {
		if c.PendingCredential == nil {
			return Enrollment{}, ErrUnauthorized
		}
		d := Device{UserID: c.UserID, Profile: Profile{Username: c.Username}, Credential: *c.PendingCredential}
		credential, err := s.verifyAssertion(c, d, raw)
		if err != nil {
			return Enrollment{}, err
		}
		d.ID = uuid.NewString()
		d.Profile.AccountID = uuid.NewString()
		d.Profile.CallID = hex.EncodeToString(randomBytes(16))
		d.Credential = credential
		session := newSession(d.Profile)
		session.AuthHandle = RandomValue()
		digest, _ := Digest(session.AuthHandle)
		return Enrollment{Device: d, Session: session, HandleDigest: digest}, nil
	})
}
func (s *Service) FinishAuthentication(ctx context.Context, id string, raw json.RawMessage) (Session, error) {
	if _, err := DecodeValue(id); err != nil {
		return Session{}, ErrUnauthorized
	}
	return s.repo.Authenticate(ctx, id, func(c Ceremony, d Device) (webauthn.Credential, Session, error) {
		credential, err := s.verifyAssertion(c, d, raw)
		if err != nil {
			return webauthn.Credential{}, Session{}, err
		}
		return credential, newSession(d.Profile), nil
	})
}
func (s *Service) verifyAssertion(c Ceremony, d Device, raw json.RawMessage) (webauthn.Credential, error) {
	var standard protocol.CredentialAssertionResponse
	if strictCredential(raw, &standard) != nil {
		return webauthn.Credential{}, ErrUnauthorized
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(raw)
	if err != nil {
		return webauthn.Credential{}, ErrUnauthorized
	}
	credential, err := s.verifier.ValidateLogin(d, c.Data, parsed)
	if err != nil || credential == nil {
		return webauthn.Credential{}, ErrUnauthorized
	}
	if !acceptable(*credential) || !parsed.Response.AuthenticatorData.Flags.HasUserVerified() {
		return webauthn.Credential{}, ErrUnauthorized
	}
	return *credential, nil
}
func acceptable(c webauthn.Credential) bool {
	return c.Flags.UserPresent && c.Flags.UserVerified && !c.Flags.BackupEligible && !c.Flags.BackupState && !c.Authenticator.CloneWarning
}
func (s *Service) Profile(ctx context.Context, token string) (Profile, error) {
	digest, err := Digest(token)
	if err != nil {
		return Profile{}, ErrUnauthorized
	}
	return s.repo.Profile(ctx, digest)
}
func (s *Service) Logout(ctx context.Context, token string) error {
	digest, err := Digest(token)
	if err != nil {
		return ErrUnauthorized
	}
	return s.repo.Logout(ctx, digest)
}

// RunCleanup is tied to the server context; each bounded sweep has its own deadline.
func (s *Service) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		sweep, cancel := context.WithTimeout(ctx, OperationTimeout)
		_ = s.repo.Cleanup(sweep) // No credential-bearing driver errors are logged.
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// The library owns protocol verification; this extra decoding rejects unknown
// JSON fields before the standard verifier parses the credential response.
func strictCredential(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
