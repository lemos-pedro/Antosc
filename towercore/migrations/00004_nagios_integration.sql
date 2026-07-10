-- +goose Up
-- +goose StatementBegin

ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS nagios_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE towers
    ADD COLUMN IF NOT EXISTS nagios_hostname TEXT NOT NULL DEFAULT '';

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS data_source TEXT NOT NULL DEFAULT 'direct_snmp';

ALTER TABLE metrics
    ADD COLUMN IF NOT EXISTS data_source TEXT NOT NULL DEFAULT 'direct_snmp';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'events_data_source_check'
    ) THEN
        ALTER TABLE events
            ADD CONSTRAINT events_data_source_check
            CHECK (data_source IN (
                'direct_snmp',
                'nagios_proxy',
                'neteco_proxy',
                'comap_proxy'
            ));
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'metrics_data_source_check'
    ) THEN
        ALTER TABLE metrics
            ADD CONSTRAINT metrics_data_source_check
            CHECK (data_source IN (
                'direct_snmp',
                'nagios_proxy',
                'neteco_proxy',
                'comap_proxy'
            ));
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_towers_nagios_hostname
    ON towers(nagios_hostname)
    WHERE nagios_enabled = TRUE;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_towers_nagios_hostname;

ALTER TABLE metrics
    DROP CONSTRAINT IF EXISTS metrics_data_source_check;

ALTER TABLE events
    DROP CONSTRAINT IF EXISTS events_data_source_check;

ALTER TABLE metrics
    DROP COLUMN IF EXISTS data_source;

ALTER TABLE events
    DROP COLUMN IF EXISTS data_source;

ALTER TABLE towers
    DROP COLUMN IF EXISTS nagios_hostname;

ALTER TABLE towers
    DROP COLUMN IF EXISTS nagios_enabled;

-- +goose StatementEnd