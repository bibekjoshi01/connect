-- +goose Up
CREATE SCHEMA connect;
COMMENT ON SCHEMA connect IS 'Connect application-owned namespace; identity tables are deferred';

-- +goose Down
-- RESTRICT is intentional: never cascade-delete future application tables.
DROP SCHEMA connect RESTRICT;
