-- +goose Up
-- +goose StatementBegin
ALTER TABLE towers
    ADD COLUMN nagios_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN nagios_hostname TEXT NOT NULL DEFAULT '';

ALTER TABLE events
    ADD COLUMN data_source TEXT NOT NULL DEFAULT 'direct_snmp'
        CHECK (data_source IN ('direct_snmp','nagios_proxy','neteco_proxy','comap_proxy'));

ALTER TABLE metrics
    ADD COLUMN data_source TEXT NOT NULL DEFAULT 'direct_snmp'
        CHECK (data_source IN ('direct_snmp','nagios_proxy','neteco_proxy','comap_proxy'));

CREATE INDEX IF NOT EXISTS idx_towers_nagios_hostname
    ON towers(nagios_hostname) WHERE nagios_enabled = TRUE;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE metrics DROP COLUMN data_source;
ALTER TABLE events DROP COLUMN data_source;
ALTER TABLE towers DROP COLUMN nagios_hostname;
ALTER TABLE towers DROP COLUMN nagios_enabled;

goose -dir=migrations postgres "..." up