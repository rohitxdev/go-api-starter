-- Create case-insensitive text column
CREATE EXTENSION IF NOT EXISTS CITEXT;

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    role TEXT NOT NULL CHECK (role IN ('user', 'admin')) DEFAULT 'user',
    email CITEXT NOT NULL UNIQUE CHECK (LENGTH(email) <= 64),
    full_name TEXT CHECK (LENGTH(full_name) <= 64),
    date_of_birth DATE CHECK (date_of_birth >= '1900-01-01'),
    gender TEXT CHECK (gender IN ('male', 'female', 'other')),
    phone_number TEXT CHECK (LENGTH(phone_number) <= 16),
    account_status TEXT NOT NULL CHECK (
        account_status IN ('active', 'suspended', 'deleted')
    ) DEFAULT 'active',
    image_url TEXT,
    is_verified BOOL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ DEFAULT current_timestamp
);

-- Create a function to update the updated_at column
CREATE OR REPLACE FUNCTION set_updated_at_column () RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = current_timestamp; 
    RETURN NEW; 
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER set_updated_at BEFORE
UPDATE ON users FOR EACH ROW
EXECUTE FUNCTION set_updated_at_column();