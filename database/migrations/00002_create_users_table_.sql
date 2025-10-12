-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id UUID DEFAULT uuidv7() PRIMARY KEY,
    username TEXT UNIQUE
        CHECK (char_length(username) BETWEEN 4 AND 32),
    email CITEXT UNIQUE NOT NULL
        CHECK (
            char_length(email) <= 256
            AND email ~* '^[A-Za-z0-9](\.?[A-Za-z0-9_-])*@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}$'
        ),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

DROP TRIGGER IF EXISTS enforce_user_timestamps ON users;

CREATE TRIGGER enforce_user_timestamps
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE PROCEDURE enforce_timestamps();

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS enforce_user_timestamps ON users;

DROP TABLE IF EXISTS users;

-- +goose StatementEnd
