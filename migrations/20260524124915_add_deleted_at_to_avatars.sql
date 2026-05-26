-- +goose Up
-- Add soft delete support
ALTER TABLE avatars
    ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

-- Index for active records
CREATE INDEX idx_avatars_user_id_active
    ON avatars (user_id)
    WHERE deleted_at IS NULL;

-- Index for deleted records cleanup
CREATE INDEX idx_avatars_deleted_at
    ON avatars (deleted_at)
    WHERE deleted_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_avatars_user_id_active;
DROP INDEX IF EXISTS idx_avatars_deleted_at;

ALTER TABLE avatars
DROP COLUMN IF EXISTS deleted_at;
