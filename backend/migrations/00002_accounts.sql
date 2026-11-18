-- +goose Up
CREATE TABLE connect.accounts (
 id uuid PRIMARY KEY,
 username text COLLATE "C" NOT NULL UNIQUE CHECK (username ~ '^[a-z][a-z0-9_]{2,31}$'),
 call_id text COLLATE "C" NOT NULL UNIQUE CHECK (call_id ~ '^[0-9a-f]{32}$'),
 active_device_id uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE connect.devices (
 id uuid PRIMARY KEY,
 account_id uuid NOT NULL REFERENCES connect.accounts(id),
 user_handle bytea NOT NULL UNIQUE CHECK (octet_length(user_handle) = 16),
 credential_id bytea NOT NULL UNIQUE CHECK (octet_length(credential_id) BETWEEN 1 AND 1023),
 credential jsonb NOT NULL CHECK (octet_length(credential::text) <= 65536),
 auth_handle_digest bytea NOT NULL UNIQUE CHECK (octet_length(auth_handle_digest) = 32),
 UNIQUE (id, account_id)
);
ALTER TABLE connect.accounts ADD CONSTRAINT active_device_belongs_to_account
 FOREIGN KEY (active_device_id, id) REFERENCES connect.devices(id, account_id)
 DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE connect.ceremonies (
 id text COLLATE "C" PRIMARY KEY CHECK (id ~ '^[A-Za-z0-9_-]{43}$'),
 operation text NOT NULL CHECK (operation IN ('registration', 'authentication')),
 username text,
 user_handle bytea NOT NULL CHECK (octet_length(user_handle) = 16),
 device_id uuid REFERENCES connect.devices(id),
 verifier_session jsonb NOT NULL CHECK (octet_length(verifier_session::text) <= 65536),
 expires_at timestamptz NOT NULL,
 attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 3),
 consumed boolean NOT NULL DEFAULT false,
 CHECK ((operation = 'registration' AND username IS NOT NULL AND device_id IS NULL)
  OR (operation = 'authentication' AND username IS NULL))
);
CREATE INDEX ceremonies_device_pending ON connect.ceremonies(device_id, expires_at) WHERE NOT consumed AND attempts < 3;
CREATE INDEX ceremonies_expiry ON connect.ceremonies(expires_at);
CREATE TABLE connect.sessions (
 digest bytea PRIMARY KEY CHECK (octet_length(digest) = 32),
 device_id uuid NOT NULL REFERENCES connect.devices(id),
 expires_at timestamptz NOT NULL,
 revoked_at timestamptz
);
CREATE UNIQUE INDEX one_unrevoked_session_per_device ON connect.sessions(device_id) WHERE revoked_at IS NULL;
CREATE INDEX sessions_expiry ON connect.sessions(expires_at);

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM connect.accounts) OR EXISTS (SELECT 1 FROM connect.devices)
  OR EXISTS (SELECT 1 FROM connect.sessions) OR EXISTS (SELECT 1 FROM connect.ceremonies)
 THEN RAISE EXCEPTION 'Refusing to remove nonempty account tables';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE connect.accounts DROP CONSTRAINT active_device_belongs_to_account;
DROP TABLE connect.sessions;
DROP TABLE connect.ceremonies;
DROP TABLE connect.devices;
DROP TABLE connect.accounts;
