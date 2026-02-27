-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS regions (
    region_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS operators (
    operator_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    code        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS towers (
    tower_id         UUID PRIMARY KEY,
    name             TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'offline'
                        CHECK (status IN ('online','degraded','offline')),
    operator_id      UUID REFERENCES operators(operator_id) ON DELETE SET NULL,
    region_id        UUID REFERENCES regions(region_id) ON DELETE SET NULL,
    availability_30d NUMERIC(5,2) NOT NULL DEFAULT 100.00,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    event_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tower_id    UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('failure','alarm','info')),
    severity    TEXT NOT NULL CHECK (severity IN ('info','warning','critical')),
    message     TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_tower_occurred ON events(tower_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);

CREATE TABLE IF NOT EXISTS tickets (
    ticket_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tower_id        UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    event_id        UUID REFERENCES events(event_id) ON DELETE SET NULL,
    status          TEXT NOT NULL DEFAULT 'open'
                        CHECK (status IN ('open','acknowledged','closed')),
    acknowledged_at TIMESTAMPTZ,
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_tower ON tickets(tower_id);

CREATE TABLE IF NOT EXISTS metrics (
    metric_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tower_id     UUID NOT NULL REFERENCES towers(tower_id) ON DELETE CASCADE,
    collected_at TIMESTAMPTZ NOT NULL,
    values       JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_metrics_tower_collected ON metrics(tower_id, collected_at DESC);

-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS metrics;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS towers;
DROP TABLE IF EXISTS operators;
DROP TABLE IF EXISTS regions;