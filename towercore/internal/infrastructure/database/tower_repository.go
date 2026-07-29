package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/security"
)

type TowerRepository struct {
	db        *sql.DB
	secretBox *security.SecretBox
}

const queryTimeout = 3 * time.Second

func NewTowerRepository(db *sql.DB, secretBox *security.SecretBox) *TowerRepository {
	return &TowerRepository{db: db, secretBox: secretBox}
}

func (r *TowerRepository) List(ctx context.Context, filter interfaces.TowerFilter) ([]domain.Tower, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 5)
	args := make([]any, 0, 8)
	next := 1

	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", next))
		args = append(args, filter.Status)
		next++
	}
	if filter.CollectionStatus != "" {
		clauses = append(clauses, fmt.Sprintf("collection_status = $%d", next))
		args = append(args, filter.CollectionStatus)
		next++
	}
	if filter.OperatorID != "" {
		// N:N via site_operators — uma torre pode ter mais de um operador
		// (cada um com armário/equipamento próprio no mesmo mastro).
		clauses = append(clauses, fmt.Sprintf(
			"tower_id::text IN (SELECT tower_id::text FROM site_operators WHERE operator_id::text = $%d)", next,
		))
		args = append(args, filter.OperatorID)
		next++
	}
	if filter.RegionID != "" {
		clauses = append(clauses, fmt.Sprintf("region_id = $%d", next))
		args = append(args, filter.RegionID)
		next++
	}
	if filter.NetecoEnabled != nil {
		clauses = append(clauses, fmt.Sprintf("neteco_enabled = $%d", next))
		args = append(args, *filter.NetecoEnabled)
		next++
	}

	if filter.Name != "" {
		// Pesquisa parcial e case-insensitive 
		// procurar sites pelo nome, não só por ID (UUID).
		clauses = append(clauses, fmt.Sprintf("name ILIKE $%d", next))
		args = append(args, "%"+filter.Name+"%")
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM towers" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
SELECT
	tower_id::text,
	name,
	status,
	vendor,
	snmp_enabled,
	snmp_version,
	snmp_target,
	snmp_community,
	snmp_v3_user,
	snmp_auth_protocol,
	snmp_auth_password,
	snmp_priv_protocol,
	snmp_priv_password,
	COALESCE(operator_id::text, ''),
	COALESCE(region_id::text, ''),
	neteco_enabled,
	neteco_neid,
	neteco_site_name,
	battery_soc,
	battery_soh,
	battery_backup_time_h,
	battery_updated_at,
	dc_output_voltage,
	dc_load_current,
	rectifier_current,
	collection_status,
	last_collected_at,
	last_successful_at,
	last_collection_error,
	availability_30d::float8,
	updated_at,
	created_at
FROM towers` + where + fmt.Sprintf(" ORDER BY created_at ASC LIMIT $%d OFFSET $%d", next, next+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	towers := make([]domain.Tower, 0)
	for rows.Next() {
		var t domain.Tower
		var status string
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&status,
			&t.Vendor,
			&t.SNMPEnabled,
			&t.SNMPVersion,
			&t.SNMPTarget,
			&t.SNMPCommunity,
			&t.SNMPV3User,
			&t.SNMPAuthProto,
			&t.SNMPAuthPass,
			&t.SNMPPrivProto,
			&t.SNMPPrivPass,
			&t.OperatorID,
			&t.RegionID,
			&t.NetecoEnabled,
			&t.NetecoNEID,
			&t.NetecoSiteName,
			&t.BatterySOC,
			&t.BatterySOH,
			&t.BatteryBackupTimeH,
			&t.BatteryUpdatedAt,
			&t.DCOutputVoltage,
			&t.DCLoadCurrent,
			&t.RectifierCurrent,
			&t.CollectionStatus,
			&t.LastCollectedAt,
			&t.LastSuccessfulAt,
			&t.LastCollectionError,
			&t.Availability30d,
			&t.UpdatedAt,
			&t.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		t.Status = domain.TowerStatus(status)
		if err := r.decryptSecrets(&t); err != nil {
			return nil, 0, err
		}
		towers = append(towers, t)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Popula Operators (N:N) para todas as torres da página numa única
	// query adicional, evitando N+1.
	towerIDs := make([]string, len(towers))
	for i, t := range towers {
		towerIDs[i] = t.ID
	}
	opsByTower, err := r.listOperatorsForTowers(ctx, towerIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range towers {
		towers[i].Operators = opsByTower[towers[i].ID]
	}

	return towers, total, nil
}

func (r *TowerRepository) GetByID(ctx context.Context, id string) (*domain.Tower, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	tower_id::text,
	name,
	status,
	vendor,
	snmp_enabled,
	snmp_version,
	snmp_target,
	snmp_community,
	snmp_v3_user,
	snmp_auth_protocol,
	snmp_auth_password,
	snmp_priv_protocol,
	snmp_priv_password,
	COALESCE(operator_id::text, ''),
	COALESCE(region_id::text, ''),
	neteco_enabled,
	neteco_neid,
	neteco_site_name,
	battery_soc,
	battery_soh,
	battery_backup_time_h,
	battery_updated_at,
	dc_output_voltage,
	dc_load_current,
	rectifier_current,
	collection_status,
	last_collected_at,
	last_successful_at,
	last_collection_error,
	availability_30d::float8,
	updated_at,
	created_at
FROM towers
WHERE tower_id::text = $1`

	var t domain.Tower
	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.Name,
		&status,
		&t.Vendor,
		&t.SNMPEnabled,
		&t.SNMPVersion,
		&t.SNMPTarget,
		&t.SNMPCommunity,
		&t.SNMPV3User,
		&t.SNMPAuthProto,
		&t.SNMPAuthPass,
		&t.SNMPPrivProto,
		&t.SNMPPrivPass,
		&t.OperatorID,
		&t.RegionID,
		&t.NetecoEnabled,
		&t.NetecoNEID,
		&t.NetecoSiteName,
		&t.BatterySOC,
		&t.BatterySOH,
		&t.BatteryBackupTimeH,
		&t.BatteryUpdatedAt,
		&t.DCOutputVoltage,
		&t.DCLoadCurrent,
		&t.RectifierCurrent,
		&t.CollectionStatus,
		&t.LastCollectedAt,
		&t.LastSuccessfulAt,
		&t.LastCollectionError,
		&t.Availability30d,
		&t.UpdatedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrTowerNotFound
		}
		return nil, err
	}
	t.Status = domain.TowerStatus(status)
	if err := r.decryptSecrets(&t); err != nil {
		return nil, err
	}

	ops, err := r.ListOperators(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.Operators = ops

	return &t, nil
}

func (r *TowerRepository) Upsert(ctx context.Context, tower *domain.Tower) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO towers (
	tower_id, name, status, vendor, snmp_enabled, snmp_version, snmp_target, snmp_community, snmp_v3_user, snmp_auth_protocol, snmp_auth_password, snmp_priv_protocol, snmp_priv_password, operator_id, region_id,
	neteco_enabled, neteco_neid, neteco_site_name,
	battery_soc, battery_soh, battery_backup_time_h, battery_updated_at,
	dc_output_voltage, dc_load_current, rectifier_current,
	collection_status, last_collected_at, last_successful_at, last_collection_error,
	availability_30d, updated_at, created_at
) VALUES (
	$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULLIF($14, '')::uuid, NULLIF($15, '')::uuid,
	$16, $17, $18,
	$19, $20, $21, $22,
	$23, $24, $25,
	$26, $27, $28, $29,
	$30, $31, $32
)
ON CONFLICT (tower_id) DO UPDATE SET
	name = EXCLUDED.name,
	status = EXCLUDED.status,
	vendor = EXCLUDED.vendor,
	snmp_enabled = EXCLUDED.snmp_enabled,
	snmp_version = EXCLUDED.snmp_version,
	snmp_target = EXCLUDED.snmp_target,
	snmp_community = EXCLUDED.snmp_community,
	snmp_v3_user = EXCLUDED.snmp_v3_user,
	snmp_auth_protocol = EXCLUDED.snmp_auth_protocol,
	snmp_auth_password = EXCLUDED.snmp_auth_password,
	snmp_priv_protocol = EXCLUDED.snmp_priv_protocol,
	snmp_priv_password = EXCLUDED.snmp_priv_password,
	operator_id = EXCLUDED.operator_id,
	region_id = EXCLUDED.region_id,
	neteco_enabled = EXCLUDED.neteco_enabled,
	neteco_neid = EXCLUDED.neteco_neid,
	neteco_site_name = EXCLUDED.neteco_site_name,
	battery_soc = EXCLUDED.battery_soc,
	battery_soh = EXCLUDED.battery_soh,
	battery_backup_time_h = EXCLUDED.battery_backup_time_h,
	battery_updated_at = EXCLUDED.battery_updated_at,
	dc_output_voltage = EXCLUDED.dc_output_voltage,
	dc_load_current = EXCLUDED.dc_load_current,
	rectifier_current = EXCLUDED.rectifier_current,
	collection_status = EXCLUDED.collection_status,
	last_collected_at = EXCLUDED.last_collected_at,
	last_successful_at = EXCLUDED.last_successful_at,
	last_collection_error = EXCLUDED.last_collection_error,
	availability_30d = EXCLUDED.availability_30d,
	updated_at = EXCLUDED.updated_at`

	encCommunity, err := r.encryptSecret(tower.SNMPCommunity)
	if err != nil {
		return err
	}
	encAuthPass, err := r.encryptSecret(tower.SNMPAuthPass)
	if err != nil {
		return err
	}
	encPrivPass, err := r.encryptSecret(tower.SNMPPrivPass)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx,
		query,
		tower.ID,
		tower.Name,
		string(tower.Status),
		tower.Vendor,
		tower.SNMPEnabled,
		tower.SNMPVersion,
		tower.SNMPTarget,
		encCommunity,
		tower.SNMPV3User,
		tower.SNMPAuthProto,
		encAuthPass,
		tower.SNMPPrivProto,
		encPrivPass,
		tower.OperatorID,
		tower.RegionID,
		tower.NetecoEnabled,
		tower.NetecoNEID,
		tower.NetecoSiteName,
		tower.BatterySOC,
		tower.BatterySOH,
		tower.BatteryBackupTimeH,
		tower.BatteryUpdatedAt,
		tower.DCOutputVoltage,
		tower.DCLoadCurrent,
		tower.RectifierCurrent,
		tower.CollectionStatus,
		tower.LastCollectedAt,
		tower.LastSuccessfulAt,
		tower.LastCollectionError,
		tower.Availability30d,
		tower.UpdatedAt,
		tower.CreatedAt,
	)
	return err
}

// listOperatorsForTowers popula os operadores (N:N via site_operators) para
// um conjunto de torres numa única query, evitando N+1 em List().
func (r *TowerRepository) listOperatorsForTowers(ctx context.Context, towerIDs []string) (map[string][]domain.Operator, error) {
	result := make(map[string][]domain.Operator)
	if len(towerIDs) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT so.tower_id::text, o.operator_id::text, o.name, o.code
FROM site_operators so
JOIN operators o ON o.operator_id = so.operator_id
WHERE so.tower_id::text = ANY($1)
ORDER BY o.name`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(towerIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var towerID string
		var op domain.Operator
		if err := rows.Scan(&towerID, &op.OperatorID, &op.Name, &op.Code); err != nil {
			return nil, err
		}
		result[towerID] = append(result[towerID], op)
	}
	return result, rows.Err()
}

// ListOperators devolve os operadores associados a uma única torre.
func (r *TowerRepository) ListOperators(ctx context.Context, towerID string) ([]domain.Operator, error) {
	m, err := r.listOperatorsForTowers(ctx, []string{towerID})
	if err != nil {
		return nil, err
	}
	return m[towerID], nil
}

// AddOperator associa um operador à torre. Idempotente (ON CONFLICT DO NOTHING).
func (r *TowerRepository) AddOperator(ctx context.Context, towerID, operatorID string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO site_operators (tower_id, operator_id)
VALUES ($1::uuid, $2::uuid)
ON CONFLICT (tower_id, operator_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query, towerID, operatorID)
	return err
}

// RemoveOperator remove a associação torre/operador (não apaga o operador).
func (r *TowerRepository) RemoveOperator(ctx context.Context, towerID, operatorID string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
DELETE FROM site_operators
WHERE tower_id::text = $1 AND operator_id::text = $2`

	res, err := r.db.ExecContext(ctx, query, towerID, operatorID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return interfaces.ErrOperatorNotFound
	}
	return nil
}

func (r *TowerRepository) encryptSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if r.secretBox == nil {
		return value, nil
	}
	return r.secretBox.Encrypt(value)
}

func (r *TowerRepository) decryptSecrets(t *domain.Tower) error {
	var err error
	if r.secretBox == nil {
		return nil
	}

	t.SNMPCommunity, err = r.secretBox.Decrypt(t.SNMPCommunity)
	if err != nil {
		return err
	}
	t.SNMPAuthPass, err = r.secretBox.Decrypt(t.SNMPAuthPass)
	if err != nil {
		return err
	}
	t.SNMPPrivPass, err = r.secretBox.Decrypt(t.SNMPPrivPass)
	if err != nil {
		return err
	}
	return nil
}
