-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
    id UUID DEFAULT uuidv7() PRIMARY KEY,
    name TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

DROP TRIGGER IF EXISTS enforce_account_timestamps ON accounts;

CREATE TRIGGER enforce_account_timestamps
BEFORE UPDATE ON accounts
FOR EACH ROW
EXECUTE PROCEDURE enforce_timestamps();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS enforce_account_timestamps ON accounts;

DROP TABLE accounts;
-- +goose StatementEnd
