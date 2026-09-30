-- +goose Up
-- Última leitura de telemetria ComAp por torre. Uma linha por torre
-- (upsert a cada ciclo do ComapScheduler), não histórico — para histórico
-- usar futuramente o mesmo padrão de /metrics ou uma tabela _history.
ALTER TABLE comap_readings
    ADD COLUMN IF NOT EXISTS engine_temp_c DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS frequency_hz DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS rpm DOUBLE PRECISION,               -- StatusStrong no profile — não fotografado no ecrã ainda
    ADD COLUMN IF NOT EXISTS voltage_l1n_v DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS voltage_l2n_v DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS voltage_l3n_v DOUBLE PRECISION,     -- fonte Mains/Gen ainda não confirmada
    ADD COLUMN IF NOT EXISTS mains_connected BOOLEAN,            -- derivado do log de eventos MCB, não do Modbus
    ADD COLUMN IF NOT EXISTS alarms_active INTEGER[];            -- índices de registos 90-138 != 0

COMMENT ON COLUMN comap_readings.battery_voltage_v IS 'StatusConfirmed no profile';
COMMENT ON COLUMN comap_readings.fuel_percent IS 'StatusSensorUnavailable no profile — sensor não instalado, não calcular';

-- +goose Down
ALTER TABLE comap_readings
    DROP COLUMN IF EXISTS engine_temp_c,
    DROP COLUMN IF EXISTS frequency_hz,
    DROP COLUMN IF EXISTS rpm,
    DROP COLUMN IF EXISTS voltage_l1n_v,
    DROP COLUMN IF EXISTS voltage_l2n_v,
    DROP COLUMN IF EXISTS voltage_l3n_v,
    DROP COLUMN IF EXISTS mains_connected,
    DROP COLUMN IF EXISTS alarms_active;