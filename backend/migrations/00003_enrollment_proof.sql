-- +goose Up
ALTER TABLE connect.ceremonies ADD COLUMN pending_credential jsonb
 CHECK (pending_credential IS NULL OR octet_length(pending_credential::text) <= 65536);
ALTER TABLE connect.ceremonies DROP CONSTRAINT ceremonies_operation_check;
ALTER TABLE connect.ceremonies ADD CONSTRAINT ceremonies_operation_check
 CHECK (operation IN ('registration', 'registration_proof', 'authentication'));
ALTER TABLE connect.ceremonies DROP CONSTRAINT ceremonies_check;
ALTER TABLE connect.ceremonies ADD CONSTRAINT ceremonies_check CHECK (
 (operation IN ('registration', 'registration_proof') AND username IS NOT NULL AND device_id IS NULL)
 OR (operation = 'authentication' AND username IS NULL)
);
ALTER TABLE connect.ceremonies ADD CONSTRAINT ceremonies_proof_state_check CHECK (
 (operation = 'registration_proof' AND pending_credential IS NOT NULL)
 OR (operation <> 'registration_proof' AND pending_credential IS NULL)
);
-- Sessions created before proof enforcement must not retain authority. Existing
-- enrolled credentials can authenticate through the normal signed assertion flow.
-- Stop the API before this explicit migration; startup never applies migrations.
UPDATE connect.sessions SET revoked_at=clock_timestamp() WHERE revoked_at IS NULL;

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM connect.accounts) OR EXISTS (SELECT 1 FROM connect.devices)
  OR EXISTS (SELECT 1 FROM connect.sessions) OR EXISTS (SELECT 1 FROM connect.ceremonies)
 THEN RAISE EXCEPTION 'Refusing to downgrade authentication with identity or ceremony data';
 END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE connect.ceremonies DROP CONSTRAINT ceremonies_proof_state_check;
ALTER TABLE connect.ceremonies DROP CONSTRAINT ceremonies_check;
ALTER TABLE connect.ceremonies DROP CONSTRAINT ceremonies_operation_check;
ALTER TABLE connect.ceremonies DROP COLUMN pending_credential;
ALTER TABLE connect.ceremonies ADD CONSTRAINT ceremonies_operation_check
 CHECK (operation IN ('registration', 'authentication'));
ALTER TABLE connect.ceremonies ADD CONSTRAINT ceremonies_check CHECK (
 (operation = 'registration' AND username IS NOT NULL AND device_id IS NULL)
 OR (operation = 'authentication' AND username IS NULL)
);
