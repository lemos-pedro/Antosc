-- +goose Up
-- Tabela de histórico para métricas de interfaces de backhaul
-- Armazena medições periódicas de status e performance de links de backhaul
CREATE TABLE IF NOT EXISTS backhaul_interface_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Identificação do site
    tower_id UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,

    -- Identificação da interface (se houver múltiplos links de backhaul)
    interface_name VARCHAR(64) NOT NULL, -- ex: "eth0", "mgmt0", "bundle-ether1"
    interface_description VARCHAR(255),  -- Descrição textual da interface (opcional)

    -- Timestamp da medição
    measured_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- === STATUS DA INTERFACE ===
    admin_status VARCHAR(16) NOT NULL,   -- up/down/testing/unknown (do IF-MIB ifAdminStatus)
    oper_status  VARCHAR(16) NOT NULL,   -- up/down/testing/unknown/dormant/notPresent/downLowerLayer (do IF-MIB ifOperStatus)
    last_change  TIMESTAMPTZ,           -- Quando a interface mudou estado pela última vez (ifLastChange)

    -- === TIPO E CAPACIDADE ===
    if_type VARCHAR(32),                 -- Tipo da interface (ianaIfType: ethernetCsmacd(6), ieee8023adLag(137), etc.)
    if_speed_bigint BIGINT,              -- Speed in bits per pixel (ifHighSpeed from IF-MIB, in Mbps * 1,000,000)
    if_speed_mbps DOUBLE PRECISION,      -- Speed in Mbps (calculado ou direto do equipamento)
    duplex_mode VARCHAR(16),             -- full/half/unknown
    media_type VARCHAR(32),              -- Tipo de meio: fiber, copper, wireless, etc.
    connector_type VARCHAR(32),          -- Tipo de conector: LC, SC, RJ45, etc.

    -- === CONTADORES DE PACOTES E BYTES (IF-MIB) ===
    in_octets BIGINT,                    -- Bytes recebidos (ifInOctets)
    out_octets BIGINT,                   -- Bytes enviados (ifOutOctets)
    in_unicast_pkts BIGINT,              -- Pacotes unicast recebidos (ifInUcastPkts)
    out_unicast_pkts BIGINT,             -- Pacotes unicast enviados (ifOutUcastPkts)
    in_discards BIGINT,                  -- Pacotes descartados na entrada (ifInDiscards)
    out_discards BIGINT,                 -- Pacotes descartados na saída (ifOutDiscards)
    in_errors BIGINT,                    -- Erros na entrada (ifInErrors)
    out_errors BIGINT,                   -- Erros na saída (ifOutErrors)
    in_unknown_protos BIGINT,            -- Pacotes descartados por protocolo desconhecido (ifInUnknownProtos)

    -- === CONTADORES DETALHADOS (ERROS DE CAMADA 2, SE DISPONÍVEL) ===
    -- Estes podem vir de MIBs específicas como EtherLike-MIB ou interfaces proprietárias
    in_frame_errors BIGINT,              -- Erros de frame (alignment, CRC, etc.)
    out_frame_errors BIGINT,
    in_jabbers BIGINT,                   -- Pacotes muito grandes com CRC ruim
    out_jabbers BIGINT,
    in_fragments BIGINT,                 -- Pacotes muito pequenos
    out_fragments BIGINT,

    -- === UTILIZAÇÃO E PERFORMANCE (CALCULADO OU MEDIDO ATIVAMENTE) ===
    utilization_pct DOUBLE PRECISION,    -- Utilização da banda em % (calculado a partir dos contadores ou medido ativamente)
    in_bandwidth_mbps DOUBLE PRECISION,  -- Banda larga de entrada em Mbps
    out_bandwidth_mbps DOUBLE PRECISION, -- Banda larga de saída em Mbps

    -- === MEDIÇÕES ATIVAS DE LATÊNCIA / LOSS / JITTER (OPCIONAL) ===
    -- Estas podem vir de probes ativos (ICMP, TCP, UDP, etc.) executados pelo próprio equipamento ou por um agente local
    avg_latency_ms DOUBLE PRECISION,     -- Latência média de ida e volta em milissegundos
    min_latency_ms DOUBLE PRECISION,
    max_latency_ms DOUBLE PRECISION,
    loss_pct DOUBLE PRECISION,           -- Percentual de perda de pacotes
    jitter_ms DOUBLE PRECISION,          -- Variação na latência (jitter)

    -- === METADADOS ===
    source_poller VARCHAR(64) NOT NULL,  -- Quem coletou estes dados (ex: "local_agent", "snmp_poller", "script")
    collection_interval_sec INTEGER,     -- Intervalo em segundos desta medição
    raw_data JSONB,                      -- Dados brutos originais (para debug/audit)

    -- Índices para consultas comuns
    CONSTRAINT chk_admin_status CHECK (admin_status IN ('up', 'down', 'testing', 'unknown')),
    CONSTRAINT chk_oper_status CHECK (oper_status IN ('up', 'down', 'testing', 'unknown', 'dormant', 'notPresent', 'downLowerLayer')),
    CONSTRAINT chk_duplex_mode CHECK (duplex_mode IN ('full', 'half', 'unknown')),
    CONSTRAINT chk_media_type CHECK (media_type IN ('fiber', 'copper', 'wireless', 'unknown'))
);

CREATE INDEX IF NOT EXISTS idx_backhaul_tower_time ON backhaul_interface_history(tower_id, measured_at DESC);
CREATE INDEX IF NOT EXISTS idx_backhaul_interface ON backhaul_interface_history(tower_id, interface_name);
CREATE INDEX IF NOT EXISTS idx_backhaul_oper_status ON backhaul_interface_history(oper_status) WHERE oper_status != 'up';
CREATE INDEX IF NOT EXISTS idx_backhaul_utilization ON backhaul_interface_history(utilization_pct);
CREATE INDEX IF NOT EXISTS idx_backhaul_latency ON backhaul_interface_history(avg_latency_ms);
CREATE INDEX IF NOT EXISTS idx_backhaul_loss ON backhaul_interface_history(loss_pct);

-- +goose Down
DROP TABLE IF EXISTS backhaul_interface_history;