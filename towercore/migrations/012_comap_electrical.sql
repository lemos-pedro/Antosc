-- +goose Up
-- Adicionar métricas elétricas do gerador à tabela comap_readings
-- (Profile atualizado em internal/adapters/comap/profile.go — inclui
-- frequência, correntes de fase e tensões fase-fase)
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS frequency_hz double precision;    -- Frequência do gerador em Hz
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS current_l1_a double precision;    -- Corrente L1 em Amperes
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS current_l2_a double precision;    -- Corrente L2 em Amperes
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS current_l3_a double precision;    -- Corrente L3 em Amperes
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS voltage_l1_l2_v double precision; -- StatusUnconfirmed no profile — mostrar com aviso no frontend
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS voltage_l2_l3_v double precision; -- StatusUnconfirmed no profile — mostrar com aviso no frontend
ALTER TABLE comap_readings
ADD COLUMN IF NOT EXISTS voltage_l3_l1_v double precision; -- StatusUnconfirmed no profile — mostrar com aviso no frontend

-- Índices para consultas comuns (alertas de sobre/under frequência, desbalanceamento de correntes)
CREATE INDEX IF NOT EXISTS idx_comap_readings_frequency ON comap_readings(frequency_hz);
CREATE INDEX IF NOT EXISTS idx_comap_readings_currents ON comap_readings(current_l1_a, current_l2_a, current_l3_a);

-- +goose Down
DROP INDEX IF EXISTS idx_comap_readings_frequency;
DROP INDEX IF EXISTS idx_comap_readings_currents;

ALTER TABLE comap_readings
DROP COLUMN IF EXISTS frequency_hz,
DROP COLUMN IF EXISTS current_l1_a,
DROP COLUMN IF EXISTS current_l2_a,
DROP COLUMN IF EXISTS current_l3_a,
DROP COLUMN IF EXISTS voltage_l1_l2_v,
DROP COLUMN IF EXISTS voltage_l2_l3_v,
DROP COLUMN IF EXISTS voltage_l3_l1_v;