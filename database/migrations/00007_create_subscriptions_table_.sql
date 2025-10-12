-- +goose Up
-- +goose StatementBegin
CREATE TABLE subscriptions (
    id UUID DEFAULT uuidv7() PRIMARY KEY,
    account_id UUID NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL
        CHECK (char_length(plan_id) BETWEEN 1 AND 64),
    status TEXT NOT NULL
        CHECK (char_length(status) BETWEEN 1 AND 32),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

DROP TRIGGER IF EXISTS enforce_user_account_timestamps ON user_accounts;

CREATE TRIGGER enforce_subscription_timestamps
BEFORE UPDATE ON subscriptions
FOR EACH ROW
EXECUTE PROCEDURE enforce_timestamps();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS enforce_subscription_timestamps ON subscriptions;

DROP TABLE subscriptions;
-- +goose StatementEnd
