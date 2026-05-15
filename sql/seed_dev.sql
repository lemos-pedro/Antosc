-- Dados de desenvolvimento (nao usar em producao)

INSERT INTO regions (region_id, name)
VALUES
    ('00000000-0000-0000-0000-000000000201', 'Region-201'),
    ('00000000-0000-0000-0000-000000000202', 'Region-202')
ON CONFLICT (region_id) DO NOTHING;

INSERT INTO operators (operator_id, name, code)
VALUES
    ('00000000-0000-0000-0000-000000000101', 'Operator-101', 'OP101'),
    ('00000000-0000-0000-0000-000000000102', 'Operator-102', 'OP102')
ON CONFLICT (operator_id) DO NOTHING;

INSERT INTO towers (tower_id, name, status, vendor, snmp_enabled, snmp_version, snmp_target, snmp_community, snmp_v3_user, snmp_auth_protocol, snmp_auth_password, snmp_priv_protocol, snmp_priv_password, operator_id, region_id, availability_30d)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'Tower-001', 'online', 'eltek', true, 'v2c', '10.10.0.11', 'public', '', '', '', '', '', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000201', 99.97),
    ('00000000-0000-0000-0000-000000000002', 'Tower-002', 'degraded', 'huawei', true, 'v2c', '10.10.0.12', 'public', '', '', '', '', '', '00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000201', 98.10),
    ('00000000-0000-0000-0000-000000000003', 'Tower-003', 'offline', 'enetek', false, 'v2c', '10.10.0.13', 'public', '', '', '', '', '', '00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000202', 93.42)
ON CONFLICT (tower_id) DO NOTHING;
