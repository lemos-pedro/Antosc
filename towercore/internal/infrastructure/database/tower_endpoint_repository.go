package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/security"
)

// TowerEndpointRepository persiste tower_endpoints (migration
// 00006_tower_endpoints.sql). O campo credential é encriptado com o mesmo
// secretBox já usado por TowerRepository para community strings SNMP,
// mantendo consistência: nenhum segredo em texto puro na base de dados.
type TowerEndpointRepository struct {
	db        *sql.DB
	secretBox *security.SecretBox
}

func NewTowerEndpointRepository(db *sql.DB, secretBox *security.SecretBox) *TowerEndpointRepository {
	return &TowerEndpointRepository{db: db, secretBox: secretBox}
}

func (r *TowerEndpointRepository) List(ctx context.Context, filter interfaces.TowerEndpointFilter) ([]interfaces.TowerEndpoint, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 3)
	args := make([]any, 0, 6)
	next := 1

	if filter.TowerID != "" {
		clauses = append(clauses, fmt.Sprintf("tower_id::text = $%d", next))
		args = append(args, filter.TowerID)
		next++
	}
	if filter.EquipmentType != "" {
		clauses = append(clauses, fmt.Sprintf("equipment_type = $%d", next))
		args = append(args, filter.EquipmentType)
		next++
	}
	if filter.Enabled != nil {
		clauses = append(clauses, fmt.Sprintf("enabled = $%d", next))
		args = append(args, *filter.Enabled)
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM tower_endpoints" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	listQuery := `
SELECT
	id::text,
	tower_id::text,
	equipment_type,
	protocol,
	COALESCE(host(ip_address), ''),
	COALESCE(port, 0),
	COALESCE(slave_id, 0),
	COALESCE(community_or_credentials, ''),
	enabled
FROM tower_endpoints` + where + fmt.Sprintf(" ORDER BY created_at ASC LIMIT $%d OFFSET $%d", next, next+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]interfaces.TowerEndpoint, 0)
	for rows.Next() {
		var ep interfaces.TowerEndpoint
		if err := rows.Scan(
			&ep.ID, &ep.TowerID, &ep.EquipmentType, &ep.Protocol,
			&ep.IPAddress, &ep.Port, &ep.SlaveID, &ep.Credential, &ep.Enabled,
		); err != nil {
			return nil, 0, err
		}
		if err := r.decryptCredential(&ep); err != nil {
			return nil, 0, err
		}
		out = append(out, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *TowerEndpointRepository) GetByTowerAndType(ctx context.Context, towerID, equipmentType string) (*interfaces.TowerEndpoint, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	id::text, tower_id::text, equipment_type, protocol,
	COALESCE(host(ip_address), ''), COALESCE(port, 0), COALESCE(slave_id, 0),
	COALESCE(community_or_credentials, ''), enabled
FROM tower_endpoints
WHERE tower_id::text = $1 AND equipment_type = $2`

	var ep interfaces.TowerEndpoint
	err := r.db.QueryRowContext(ctx, query, towerID, equipmentType).Scan(
		&ep.ID, &ep.TowerID, &ep.EquipmentType, &ep.Protocol,
		&ep.IPAddress, &ep.Port, &ep.SlaveID, &ep.Credential, &ep.Enabled,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrTowerEndpointNotFound
		}
		return nil, err
	}
	if err := r.decryptCredential(&ep); err != nil {
		return nil, err
	}
	return &ep, nil
}

func (r *TowerEndpointRepository) Create(ctx context.Context, ep *interfaces.TowerEndpoint) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	encCredential, err := r.encryptCredential(ep.Credential)
	if err != nil {
		return err
	}

	const query = `
INSERT INTO tower_endpoints (
	tower_id, equipment_type, protocol, ip_address, port, slave_id,
	community_or_credentials, enabled
) VALUES (
	$1::uuid, $2, $3, NULLIF($4, '')::inet, NULLIF($5, 0), NULLIF($6, 0), $7, $8
)`

	_, err = r.db.ExecContext(ctx, query,
		ep.TowerID, ep.EquipmentType, ep.Protocol, ep.IPAddress, ep.Port,
		ep.SlaveID, encCredential, ep.Enabled,
	)
	return err
}

func (r *TowerEndpointRepository) Update(ctx context.Context, ep *interfaces.TowerEndpoint) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	encCredential, err := r.encryptCredential(ep.Credential)
	if err != nil {
		return err
	}

	const query = `
UPDATE tower_endpoints SET
	protocol = $1,
	ip_address = NULLIF($2, '')::inet,
	port = NULLIF($3, 0),
	slave_id = NULLIF($4, 0),
	community_or_credentials = $5,
	enabled = $6,
	updated_at = now()
WHERE id::text = $7`

	res, err := r.db.ExecContext(ctx, query,
		ep.Protocol, ep.IPAddress, ep.Port, ep.SlaveID, encCredential, ep.Enabled, ep.ID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return interfaces.ErrTowerEndpointNotFound
	}
	return nil
}

func (r *TowerEndpointRepository) encryptCredential(value string) (string, error) {
	if value == "" || r.secretBox == nil {
		return value, nil
	}
	return r.secretBox.Encrypt(value)
}

func (r *TowerEndpointRepository) decryptCredential(ep *interfaces.TowerEndpoint) error {
	if r.secretBox == nil || ep.Credential == "" {
		return nil
	}
	v, err := r.secretBox.Decrypt(ep.Credential)
	if err != nil {
		return err
	}
	ep.Credential = v
	return nil
}