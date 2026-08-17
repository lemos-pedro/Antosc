-- +goose Up
-- Adicionar métricas de fibra ótica essenciais para monitoring de field
-- Essas métricas permitem ao técnico de campo diagnosticar problemas físicos de link de fibra
ALTER TABLE backhaul_interface_history
ADD COLUMN IF NOT EXISTS tx_power_dbm DOUBLE PRECISION;              -- Potência de transmissão óptica em dBm
ADD COLUMN IF NOT EXISTS rx_power_dbm DOUBLE PRECISION;              -- Potência de recebimento óptico em dBm
ADD COLUMN IF NOT EXISTS optic_temp_c DOUBLE PRECISION;              -- Temperatura do transceptor óptico em °C
ADD COLUMN IF NOT EXISTS optic_bias_current_ma DOUBLE PRECISION;     -- Corrente de bias do laser em mA (indicador de envelhecimento)
ADD COLUMN IF NOT EXISTS los_events BIGINT;                          -- Contagem de eventos Loss of Signal
ADD COLUMN IF NOT EXISTS lof_events BIGINT;                          -- Contagem de eventos Loss of Frame
ADD COLUMN IF NOT EXISTS lom_events BIGINT;                          -- Contagem de eventos Loss of Multiframe
ADD COLUMN IF NOT EXISTS optic_wavelength_nm INTEGER;                -- Comprimento de onda nominal em nm (ex: 850, 1310, 1550)
ADD COLUMN IF NOT EXISTS optic_vendor VARCHAR(64);                   -- Vendor do transceptor (ex: Finisar, II-VI, Accelink)
ADD COLUMN IF NOT EXISTS optic_part_number VARCHAR(64);              -- Número de peça do transceptor (para reposição exata)
ADD COLUMN IF NOT EXISTS optic_serial_number VARCHAR(64);            -- Número de série (para rastreamento de garantia e histórico)
ADD COLUMN IF NOT EXISTS optic_date_code VARCHAR(16);                -- Data de fabricação (YYWW ou YYYYMM)

-- Índices para consultas de alertas rápidos
CREATE INDEX IF NOT EXISTS idx_backhaul_tx_power ON backhaul_interface_history(tx_power_dbm);
CREATE INDEX IF NOT EXISTS idx_backhaul_rx_power ON backhaul_interface_history(rx_power_dbm);
CREATE INDEX IF NOT EXISTS idx_backhaul_optic_temp ON backhaul_interface_history(optic_temp_c);
CREATE INDEX IF NOT EXISTS idx_backhaul_los ON backhaul_interface_history(los_events) WHERE los_events > 0;
CREATE INDEX IF NOT EXISTS idx_backhaul_lof ON backhaul_interface_history(lof_events) WHERE lof_events > 0;
CREATE INDEX IF NOT EXISTS idx_backhaul_lom ON backhaul_interface_history(lom_events) WHERE lom_events > 0;

-- +goose Down
ALTER TABLE backhaul_interface_history
DROP COLUMN IF EXISTS tx_power_dbm,
DROP COLUMN IF EXISTS rx_power_dbm,
DROP COLUMN IF EXISTS optic_temp_c,
DROP COLUMN IF EXISTS optic_bias_current_ma,
DROP COLUMN IF EXISTS los_events,
DROP COLUMN IF EXISTS lof_events,
DROP COLUMN IF EXISTS lom_events,
DROP COLUMN IF EXISTS optic_wavelength_nm,
DROP COLUMN IF EXISTS optic_vendor,
DROP COLUMN IF EXISTS optic_part_number,
DROP COLUMN IF EXISTS optic_serial_number,
DROP COLUMN IF EXISTS optic_date_code;

DROP INDEX IF EXISTS idx_backhaul_tx_power;
DROP INDEX IF EXISTS idx_backhaul_rx_power;
DROP INDEX IF EXISTS idx_backhaul_optic_temp;
DROP INDEX IF EXISTS idx_backhaul_los;
DROP INDEX IF EXISTS idx_backhaul_lof;
DROP INDEX IF EXISTS idx_backhaul_lom;