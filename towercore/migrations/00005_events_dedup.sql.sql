-- +goose Up
-- Deduplicação de eventos: um alarme persistente (ex. subtensão contínua)
-- deixa de gerar um novo registo em `events` a cada ciclo de poll (~45s).
-- Em vez disso, o evento "open" existente é reutilizado até a condição
-- deixar de se verificar.
--
-- IDEMPOTENTE: usa IF NOT EXISTS em todas as colunas/índices para poder
-- correr com segurança independentemente de já teres testado parte disto
-- manualmente via psql.

ALTER TABLE events ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'open';
ALTER TABLE events ADD COLUMN IF NOT EXISTS alarm_key VARCHAR(100);
ALTER TABLE events ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
ALTER TABLE events ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Backfill: eventos já existentes (criados antes desta migração) ficam
-- "resolved" — não sabemos o estado real retroativo, e não queremos
-- herdar milhões de "open" órfãos. Só toca em linhas sem alarm_key
-- definido, para não sobrescrever nada que já tenha sido processado
-- por uma corrida anterior desta migração.
UPDATE events
SET status = 'resolved', resolved_at = COALESCE(resolved_at, created_at)
WHERE alarm_key IS NULL AND status = 'open';

-- Só pode existir UM evento "open" por torre+alarme em simultâneo.
CREATE UNIQUE INDEX IF NOT EXISTS idx_events_open_unique
    ON events (tower_id, alarm_key)
    WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_events_tower_status ON events (tower_id, status);

-- +goose Down
DROP INDEX IF EXISTS idx_events_tower_status;
DROP INDEX IF EXISTS idx_events_open_unique;
ALTER TABLE events
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS alarm_key,
    DROP COLUMN IF EXISTS resolved_at,
    DROP COLUMN IF EXISTS last_seen_at;