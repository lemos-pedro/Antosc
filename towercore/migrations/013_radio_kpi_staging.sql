 -- +goose Up
--Tabela de staging para KPIs de rádio já processados/federados do NMS/EMS existente
-- Permite correlacionar dados de infraestrutura (TowerCore) com qualidade de serviço de rádio
CREATE TABLE IF NOT EXISTS radio_kpi_staging (
     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identificação do site/setor
    tower_id UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    sector_id VARCHAR(10), -- ex: "A", "B", "C" ou "0", "1", "2"
    cell_technique VARCHAR(10) NOT NULL, -- ex: "LTE", "5GNR", "GSM", "UMTS"

    -- Timestamp da medição
    measured_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),

   -- === MÉTRICAS DE POTÊNCIA E QUALIDADE ===
    tx_power_watt DOUBLE PRECISION,           -- Potência de transmissão em Watts
    tx_power_dbm DOUBLE PRECISION,            -- Potência de transmissão em dBm
    rx_power_dbm DOUBLE PRECISION,            -- Potência média de sinal recebido em dBm
    snr_db DOUBLE PRECISION,                  -- Signal-to-Noise Ratio em dB
    sinr_db DOUBLE PRECISION,                 -- Signal-to-Interference-plus-Noise Ratio em dB
    rsrp_dbm DOUBLE PRECISION,                -- LTE/5G: Reference Signal Received Power
    rsrq_db DOUBLE PRECISION,                 -- LTE/5G: Reference Signal Received Quality
  
   -- === MÉTRICAS DE UTILIZAÇÃO E CAPACIDADE ===
    connected_ues INTEGER,                    -- Número de User Equipments conectados
    max_supported_ues INTEGER,                -- Capacidade máxima teórica do setor
    prb_utilization_pct DOUBLE PRECISION,     -- LTE/5G: Physical Resource Block utilization (%)
    channel_occupancy_pct DOUBLE PRECISION,   -- GSM/UMTS: Channel occupancy (%)

    -- === MÉTRICAS DE ERROS E QUALIDADE ===
    ber DOUBLE PRECISION,                     -- Bit Error Rate
    bler DOUBLE PRECISION,                    -- Block Error Rate
    fer DOUBLE PRECISION,                     -- Frame Error Rate
    codec_drop_pct DOUBLE PRECISION,          -- Percentual de pacotes descartados por falha de codec
    -- === MÉTRICAS DE MOBILIDADE ===
    ho_attempt INTEGER,                       -- Número de tentativas de handover
    ho_success INTEGER,                       -- Número de handovers bem-sucedidos
    ho_fail INTEGER,                          -- Número de handovers que falharam
    ho_ping_pong INTEGER,                     -- Handovers ping-pong (setor A->B->A em curto tempo)

    -- === MÉTRICAS DE DISPONIBILIDADE E EXPERIÊNCIA ===
    call_drop_pct DOUBLE PRECISION,           -- Percentual de chamadas que caíram
    call_block_pct DOUBLE PRECISION,          -- Percentual de chamadas bloqueadas por falta de recursos
    pdcp_sdu_loss_pct DOUBLE PRECISION,       -- Perdida de pacotes na camada PDCP
    rlc_retrans_pct DOUBLE PRECISION,         -- Retransmissões na camada RLC

   -- === METADADOS ===
    source_system VARCHAR(50) NOT NULL,       -- Qual sistema NMS/EMS enviou estes dados (ex: "Ericsson NMS", "Nokia NetAct", "Custom Collector")
    collection_interval_sec INTEGER,          -- Intervalo de coleta destes KPIs (ex: 300 = 5 min)
    raw_data JSONB,                           -- Dados brutos originais para auditoria/debug

    -- Índices para consultas comuns
    CONSTRAINT chk_technique CHECK (cell_technique IN ('LTE', '5GNR', 'GSM', 'UMTS', 'NR'))
 );

CREATE INDEX IF NOT EXISTS idx_radio_kpi_tower_time ON radio_kpi_staging(tower_id, measured_at DESC);
CREATE INDEX IF NOT EXISTS idx_radio_kpi_technique ON radio_kpi_staging(cell_technique);
CREATE INDEX IF NOT EXISTS idx_radio_kpi_sector ON radio_kpi_staging(tower_id, sector_id);
CREATE INDEX IF NOT EXISTS idx_radio_kpi_source ON radio_kpi_staging(source_system);
CREATE INDEX IF NOT EXISTS idx_radio_kpi_rx_power ON radio_kpi_staging(rx_power_dbm);
CREATE INDEX IF NOT EXISTS idx_radio_kpi_prb_util ON radio_kpi_staging(prb_utilization_pct);

 -- +goose Down
 DROP TABLE IF EXISTS radio_kpi_staging;