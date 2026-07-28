-- +goose Up
ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS collection_status TEXT NOT NULL DEFAULT 'not_configured',
    ADD COLUMN IF NOT EXISTS last_collected_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_successful_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_collection_error TEXT NOT NULL DEFAULT '';

ALTER TABLE towers
    ADD CONSTRAINT towers_collection_status_check
    CHECK (collection_status IN ('not_configured', 'no_credentials', 'never_collected', 'collection_failed', 'active'));

CREATE INDEX IF NOT EXISTS idx_towers_collection_status ON towers(collection_status);

ALTER TABLE towers DROP CONSTRAINT IF EXISTS towers_status_check;
ALTER TABLE towers
    ADD CONSTRAINT towers_status_check
    CHECK (status IN ('online', 'degraded', 'offline', 'no_data'));
ALTER TABLE towers ALTER COLUMN status SET DEFAULT 'no_data';

-- +goose Down
DROP INDEX IF EXISTS idx_towers_collection_status;
ALTER TABLE towers
    DROP CONSTRAINT IF EXISTS towers_collection_status_check,
    DROP COLUMN IF EXISTS last_collection_error,
    DROP COLUMN IF EXISTS last_successful_at,
    DROP COLUMN IF EXISTS last_collected_at,
    DROP COLUMN IF EXISTS collection_status;
