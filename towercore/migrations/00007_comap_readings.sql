-- +goose Up
-- Última leitura de telemetria ComAp por torre. Uma linha por torre
-- (upsert a cada ciclo do ComapScheduler), não histórico — para histórico
-- usar futuramente o mesmo padrão de /metrics ou uma tabela _history.
CREATE TABLE IF NOT EXISTS comap_readings (
    tower_id UUID PRIMARY KEY REFERENCES towers(tower_id) ON DELETE CASCADE,
    fuel_liters DOUBLE PRECISION,
    fuel_percent DOUBLE PRECISION,       -- StatusUnconfirmed no profile — mostrar com aviso no frontend
    battery_voltage_v DOUBLE PRECISION,  -- StatusUnconfirmed no profile — mostrar com aviso no frontend
    run_hours_total DOUBLE PRECISION,
    collected_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS comap_readings;