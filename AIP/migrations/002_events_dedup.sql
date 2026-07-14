-- +goose Up

-- Padrão CreateOrTouch/Resolve para ai_events: sem isto, cada ciclo do
-- scheduler (5 em 5 min) enquanto uma torre continuar degraded/offline
-- criaria uma linha nova, explodindo a tabela. Com alarm_key + status,
-- uma condição persistente só gera UMA linha "open", que vai sendo
-- "tocada" (last_seen_at) em cada ciclo, até a torre recuperar.

ALTER TABLE ai_events ADD COLUMN IF NOT EXISTS alarm_key VARCHAR(150);
ALTER TABLE ai_events ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'open';
ALTER TABLE ai_events ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP NOT NULL DEFAULT NOW();
ALTER TABLE ai_events ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;

-- Só pode existir UM evento "open" por (torre, alarm_key) em simultâneo.
-- É esta restrição que o ON CONFLICT do CreateOrTouch usa.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_events_open_dedup
ON ai_events (tower_id, alarm_key)
WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_ai_events_status
ON ai_events (status);

-- +goose Down
DROP INDEX IF EXISTS idx_ai_events_status;
DROP INDEX IF EXISTS idx_ai_events_open_dedup;
ALTER TABLE ai_events DROP COLUMN IF EXISTS resolved_at;
ALTER TABLE ai_events DROP COLUMN IF EXISTS last_seen_at;
ALTER TABLE ai_events DROP COLUMN IF EXISTS status;
ALTER TABLE ai_events DROP COLUMN IF EXISTS alarm_key;
