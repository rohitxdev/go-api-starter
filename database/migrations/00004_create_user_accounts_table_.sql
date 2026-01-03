-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_accounts (
    user_id UUID REFERENCES users(id),
    account_id UUID REFERENCES accounts(id),
    status CITEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'pending')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, account_id)
);

DROP TRIGGER IF EXISTS enforce_user_account_timestamps ON user_accounts;

CREATE TRIGGER enforce_user_account_timestamps
BEFORE UPDATE ON user_accounts
FOR EACH ROW
EXECUTE PROCEDURE enforce_timestamps();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS enforce_user_account_timestamps ON user_accounts;

DROP TABLE user_accounts;
-- +goose StatementEnd
