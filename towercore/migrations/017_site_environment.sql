-- +goose Up
-- Tabela para monitoramento de ambiente e condições físicas do site
-- Foco em métricas acionáveis pelo técnico de campo: temperatura, umidade, porta, energia, fumaça, vibração
CREATE TABLE site_environment (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id         UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    measured_at     TIMESTAMPTZ NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- === TEMPERATURA INTERNA (CRÍTICA PARA EQUIPAMENTOS) ===
    -- Temperatura do ar dentro do shelter/cabinet
    internal_temp_c DOUBLE PRECISION,                     -- °C
    -- Temperatura em pontos críticos (ex: próximo a fontes de calor, baterias)
    temp_risk_point_c DOUBLE PRECISION,                   -- °C (opcional, para pontos específicos como gerador, baterias)
    -- Diferença entre ponto quente e ambiente interno (detecta hotspots)
    temp_delta_c DOUBLE PRECISION,                        -- °C (ex: temp_risk_point_c - internal_temp_c)

    -- === UMIDADE RELATIVA (RISCO DE CORROSÃO E CONDENSAÇÃO) ===
    humidity_pct DOUBLE PRECISION,                        -- % RH (Relative Humidity)
    -- Umidade em pontos críticos (ex: perto de entrada, piso elevado)
    humidity_risk_point_pct DOUBLE PRECISION,             -- % RH (opcional)

    -- === PORTA DO SHELTER/CABINET ===
    door_open BOOLEAN,                                    -- true = porta aberta (risco de invasão, umidade, poeira)
    -- Alguns sistemas reportam múltiplas portas (ex: principal, traseira, de baterias)
    door_open_secondary BOOLEAN,                          -- opcional

    -- === ENERGIA ===
    -- Fonte principal de energia
    mains_power_ok BOOLEAN,                               -- true = energia da rede presente e estável
    -- Estado do nobreak/UPS
    ups_on_battery BOOLEAN,                             -- true = UPS está fornecendo energia da bateria
    ups_battery_pct DOUBLE PRECISION,                     -- % de carga restante da bateria do UPS
    ups_load_pct DOUBLE PRECISION,                        -- % de carga atual do UPS
    -- Estado do gerador (se presente)
    generator_running BOOLEAN,                            -- true = gerador em funcionamento
    generator_load_pct DOUBLE PRECISION,                  -- % de carga no gerador
    generator_fuel_pct DOUBLE PRECISION,                  -- % de combustível restante (se sensor disponível)
    -- Tensão e frequência (se disponível e útil para diagnóstico)
    mains_voltage_v DOUBLE PRECISION,                     -- Volts AC (ex: 220V)
    mains_frequency_hz DOUBLE PRECISION,                  -- Hertz (ex: 60Hz)

    -- === DETECÇÃO DE FUMAÇA / INCÊNDIO ===
    smoke_detected BOOLEAN,                               -- true = fumaça detectada (alarme imediato)
    -- Alguns sistemas têm múltiplos zonas
    smoke_detected_zone_2 BOOLEAN,                        -- opcional
    smoke_detected_zone_3 BOOLEAN,                        -- opcional

    -- === VIBRAÇÃO / IMPACTO (RISCO DE DANO FÍSICO OU INVASÃO) ===
    vibration_detected BOOLEAN,                           -- true = vibração anormal (possível tentativa de arrombamento, impacto estrutural)
    -- Alguns sistemas têm sensores por zona
    vibration_severity SMALLINT,                          -- 0=none, 1=low, 2=medium, 3=high

    -- === QUALIDADE DO AR (OPCIONAL, PARA AMBIENTES ESPECIAIS) ===
    -- Concentração de partículas (poeira, poluição) -- relevante em áreas industriais ou desertos
    pm2_5_ugm3 DOUBLE PRECISION,                          -- µg/m³ (partículas finas)
    pm10_ugm3 DOUBLE PRECISION,                           -- µg/m³ (partículas grossas)
    -- Gases perigosos (ex: em bunkers ou perto de fábricas)
    co_ppm INTEGER,                                       -- partes por milhão de monóxido de carbono
    ch4_ppm INTEGER,                                      -- partes por milhão de metano (risco de explosão)

    -- === METADADOS ===
    source_poller     VARCHAR(64) NOT NULL,               -- ex: "local_agent", "snmp_poller", "modbus", "script"
    collection_interval_sec INT NOT NULL,                 -- em segundos
    raw_data          JSONB,                              -- dados brutos originais (para debug/audit)
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- === ÍNDICES PARA CONSULTAS RÁPIDAS E ALERTAS ===
-- Consultas recentes por site (mais comuns)
CREATE INDEX IF NOT EXISTS idx_site_env_site_measured ON site_environment(site_id, measured_at DESC);
-- Alertas de temperatura alta
CREATE INDEX IF NOT EXISTS idx_site_env_temp_high ON site_environment(internal_temp_c) WHERE internal_temp_c > 35;
-- Alertas de umidade alta
CREATE INDEX IF NOT EXISTS idx_site_env_humidity_high ON site_environment(humidity_pct) WHERE humidity_pct > 80;
-- Alertas de porta aberta
CREATE INDEX IF NOT EXISTS idx_site_env_door_open ON site_environment(door_open) WHERE door_open = true;
-- Alertas de energia falhando
CREATE INDEX IF NOT EXISTS idx_site_env_mains_down ON site_environment(mains_power_ok) WHERE mains_power_ok = false;
-- Alertas de UPS em bateria
CREATE INDEX IF NOT EXISTS idx_site_env_ups_battery ON site_environment(ups_on_battery) WHERE ups_on_battery = true;
-- Alertas de fumaça
CREATE INDEX IF NOT EXISTS idx_site_env_smoke ON site_environment(smoke_detected) WHERE smoke_detected = true;
-- Alertas de vibração
CREATE INDEX IF NOT EXISTS idx_site_env_vibration ON site_environment(vibration_detected) WHERE vibration_detected = true;
-- Alertas de gerador com baixa carga (possível problema)
CREATE INDEX IF NOT EXISTS idx_site_env_generator_low_load ON site_environment(generator_load_pct) WHERE generator_running = true AND generator_load_pct < 20;
-- Alertas de bateria do UPS baixa
CREATE INDEX IF NOT EXISTS idx_site_env_ups_low_battery ON site_environment(ups_battery_pct) WHERE ups_battery_pct < 20;

-- +goose Down
DROP TABLE IF EXISTS site_environment;
