-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION enforce_timestamps()
RETURNS TRIGGER AS $$
BEGIN
    -- Rule 1: Prevent updates to created_at
    IF NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'created_at cannot be updated';
    END IF;
    -- Rule 2: Automatically update updated_at on any update
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS enforce_timestamps();

DROP EXTENSION IF EXISTS citext;
-- +goose StatementEnd