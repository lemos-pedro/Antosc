package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

	clauses := make([]string, 0, 3)
	args := make([]any, 0, 8)
	next := 1

	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", next))
		args = append(args, filter.Status)
		next++
	}
	if filter.OperatorID != "" {
		clauses = append(clauses, fmt.Sprintf("operator_id = $%d", next))
		args = append(args, filter.OperatorID)
		next++
	}
	if filter.RegionID != "" {
		clauses = append(clauses, fmt.Sprintf("region_id = $%d", next))
		args = append(args, filter.RegionID)
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
	return &t, nil
}

func (r *TowerRepository) Upsert(ctx context.Context, tower *domain.Tower) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO towers (
	tower_id, name, status, vendor, snmp_enabled, snmp_version, snmp_target, snmp_community, snmp_v3_user, snmp_auth_protocol, snmp_auth_password, snmp_priv_protocol, snmp_priv_password, operator_id, region_id, availability_30d, updated_at, created_at
) VALUES (
	$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NULLIF($14, '')::uuid, NULLIF($15, '')::uuid, $16, $17, $18
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
		tower.Availability30d,
		tower.UpdatedAt,
		tower.CreatedAt,
	)
	return err
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
