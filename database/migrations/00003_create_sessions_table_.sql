-- +goose Up
-- +goose StatementBegin
CREATE TABLE sessions (
    id UUID DEFAULT uuidv7() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_agent TEXT
        CHECK (char_length(user_agent) BETWEEN 4 AND 256),
    ip_address INET NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);


DROP TRIGGER IF EXISTS enforce_session_timestamps on sessions;

CREATE TRIGGER enforce_session_timestamps
BEFORE UPDATE ON sessions
FOR EACH ROW
EXECUTE PROCEDURE enforce_timestamps();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS enforce_session_timestamps ON sessions;

DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd
