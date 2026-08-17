-- +goose Up
-- Colunas de status de backhaul para valor atual e alertas rápidos
-- Armazenam o estado mais recente conhecido da interface de backhaul principal
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_if_name VARCHAR(64);                    -- Nome da interface (ex: "eth0", "bundle-ether1")
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_if_description VARCHAR(255);            -- Descrição textual
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_admin_status VARCHAR(16);               -- up/down/etc
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_oper_status VARCHAR(16);                -- up/down/etc (CRÍTICO para alertas)
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_last_change TIMESTAMPTZ;                -- Quando mudou estado pela última vez
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_if_type VARCHAR(32);                   -- Tipo de interface
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_if_speed_mbps DOUBLE PRECISION;         -- Velocidade em Mbps
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_duplex_mode VARCHAR(16);               -- full/half/unknown
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_media_type VARCHAR(32);                -- fiber/copper/wireless/unknown
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_media_connector VARCHAR(32);           -- Tipo de conector (LC, SC, RJ45, etc.)
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_in_octets BIGINT;                     -- Bytes recebidos (contador)
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_out_octets BIGINT;                    -- Bytes enviados (contador)
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_in_errors BIGINT;                    -- Erros de entrada
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_out_errors BIGINT;                   -- Erros de saída
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_in_discards BIGINT;                  -- Pacotes descartados na entrada
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_out_discards BIGINT;                 -- Pacotes descartados na saída
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_utilization_pct DOUBLE PRECISION;     -- Utilização % (calculado)
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_avg_latency_ms DOUBLE PRECISION;      -- Latência média (ms)
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_loss_pct DOUBLE PRECISION;            -- Perda de pacotes %
ALTER TABLE towers
ADD COLUMN IF NOT EXISTS backhaul_last_updated TIMESTAMPTZ;              -- Quando este registro foi atualizado pela última vez

-- Índices para alertas e consultas rápidas
CREATE INDEX IF NOT EXISTS idx_towers_backhaul_oper_down ON towers(backhaul_oper_status) WHERE backhaul_oper_status != 'up';
CREATE INDEX IF NOT EXISTS idx_towers_backhaul_high_util ON towers(backhaul_utilization_pct) WHERE backhaul_utilization_pct > 80;
CREATE INDEX IF NOT EXISTS idx_towers_backhaul_high_latency ON towers(backhaul_avg_latency_ms) WHERE backhaul_avg_latency_ms > 50;
CREATE INDEX IF NOT EXISTS idx_towers_backhaul_high_loss ON towers(backhaul_loss_pct) WHERE backhaul_loss_pct > 1.0;

-- +goose Down
ALTER TABLE towers
DROP COLUMN IF EXISTS backhaul_if_name,
DROP COLUMN IF EXISTS backhaul_if_description,
DROP COLUMN IF EXISTS backhaul_admin_status,
DROP COLUMN IF EXISTS backhaul_oper_status,
DROP COLUMN IF EXISTS backhaul_last_change,
DROP COLUMN IF EXISTS backhaul_if_type,
DROP COLUMN IF EXISTS backhaul_if_speed_mbps,
DROP COLUMN IF EXISTS backhaul_duplex_mode,
DROP COLUMN IF EXISTS backhaul_media_type,
DROP COLUMN IF EXISTS backhaul_media_connector,
DROP COLUMN IF EXISTS backhaul_in_octets,
DROP COLUMN IF EXISTS backhaul_out_octets,
DROP COLUMN IF EXISTS backhaul_in_errors,
DROP COLUMN IF EXISTS backhaul_out_errors,
DROP COLUMN IF EXISTS backhaul_in_discards,
DROP COLUMN IF EXISTS backhaul_out_discards,
DROP COLUMN IF EXISTS backhaul_utilization_pct,
DROP COLUMN IF EXISTS backhaul_avg_latency_ms,
DROP COLUMN IF EXISTS backhaul_loss_pct,
DROP COLUMN IF EXISTS backhaul_last_updated;

DROP INDEX IF EXISTS idx_towers_backhaul_oper_down;
DROP INDEX IF EXISTS idx_towers_backhaul_high_util;
DROP INDEX IF EXISTS idx_towers_backhaul_high_latency;
DROP INDEX IF EXISTS idx_towers_backhaul_high_loss;