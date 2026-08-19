-- +goose Up

CREATE TABLE network_links (
    link_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    router_host         TEXT NOT NULL,
    router_ip           TEXT NOT NULL,
    if_index            INTEGER NOT NULL,
    if_descr            TEXT,
    if_alias            TEXT,
    operator            TEXT,
    media_type          TEXT NOT NULL DEFAULT 'fiber' CHECK (media_type IN ('fiber', 'radio')),
    nominal_capacity_mb BIGINT NOT NULL DEFAULT 0,
    region_id           UUID REFERENCES regions(region_id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (router_ip, if_index)
);

CREATE INDEX idx_network_links_router_ip ON network_links (router_ip);
CREATE INDEX idx_network_links_operator ON network_links (operator);

CREATE TABLE link_metric_snapshots (
    snapshot_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id          UUID NOT NULL REFERENCES network_links(link_id) ON DELETE CASCADE,
    collected_at     TIMESTAMPTZ NOT NULL,
    oper_status      TEXT NOT NULL,
    admin_status     TEXT NOT NULL,
    in_octets        BIGINT NOT NULL DEFAULT 0,
    out_octets       BIGINT NOT NULL DEFAULT 0,
    in_errors        BIGINT NOT NULL DEFAULT 0,
    out_errors       BIGINT NOT NULL DEFAULT 0,
    in_discards      BIGINT NOT NULL DEFAULT 0,
    out_discards     BIGINT NOT NULL DEFAULT 0,
    speed_mb         BIGINT NOT NULL DEFAULT 0,
    in_util_percent  DOUBLE PRECISION,
    out_util_percent DOUBLE PRECISION
);

-- Índice para GetLastSnapshot (busca mais recente por link) e para
-- consultas históricas de qualidade/utilização por período.
CREATE INDEX idx_link_metric_snapshots_link_collected
    ON link_metric_snapshots (link_id, collected_at DESC);

CREATE TABLE link_events (
    event_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id     UUID NOT NULL REFERENCES network_links(link_id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    severity    TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    message     TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    alarm_key   TEXT NOT NULL,
    UNIQUE (alarm_key, resolved_at)
);

CREATE INDEX idx_link_events_link_id ON link_events (link_id);
CREATE INDEX idx_link_events_alarm_key_open
    ON link_events (alarm_key) WHERE resolved_at IS NULL;

-- +goose Down

DROP TABLE IF EXISTS link_events;
DROP TABLE IF EXISTS link_metric_snapshots;
DROP TABLE IF EXISTS network_links;
