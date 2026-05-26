-- +goose Up
CREATE TABLE IF NOT EXISTS avatars (
                                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    original_url TEXT NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    content_type VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'uploading',
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_avatars_user_id ON avatars(user_id);
CREATE INDEX IF NOT EXISTS idx_avatars_status ON avatars(status);
CREATE INDEX IF NOT EXISTS idx_avatars_created_at ON avatars(created_at);

-- +goose Down
DROP TABLE IF EXISTS avatars;
