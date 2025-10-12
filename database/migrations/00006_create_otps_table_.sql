-- +goose Up
-- +goose StatementBegin
CREATE TABLE otps (
    id UUID DEFAULT uuidv7() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash BYTEA NOT NULL
        CHECK (octet_length(code_hash) BETWEEN 8 AND 512),
    attempts INT DEFAULT 0 NOT NULL,
    consumed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

DROP TRIGGER IF EXISTS enforce_user_account_timestamps ON user_accounts;

CREATE TRIGGER enforce_otp_timestamps
BEFORE UPDATE ON otps
FOR EACH ROW
EXECUTE PROCEDURE enforce_timestamps();

CREATE INDEX IF NOT EXISTS idx_otps_user_id ON otps(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_otps_user_id;

DROP TRIGGER IF EXISTS enforce_otp_timestamps ON otps;

DROP TABLE otps;
-- +goose StatementEnd
