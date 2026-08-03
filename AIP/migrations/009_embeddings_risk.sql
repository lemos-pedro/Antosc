-- ==========================================
-- 009: TOWER EMBEDDINGS (cross-tower)
-- ==========================================

CREATE TABLE IF NOT EXISTS tower_embeddings (
    tower_id VARCHAR(64) PRIMARY KEY,
    dims INT NOT NULL,
    vector DOUBLE PRECISION[] NOT NULL,
    model_version VARCHAR(32) NOT NULL DEFAULT 'pca-v1',
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tower_embeddings_updated ON tower_embeddings (updated_at DESC);
