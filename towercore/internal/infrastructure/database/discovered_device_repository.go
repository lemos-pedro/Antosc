package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/security"
)

type DiscoveredDeviceRepository struct {
	db        *sql.DB
	secretBox *security.SecretBox
}

func NewDiscoveredDeviceRepository(db *sql.DB, secretBox *security.SecretBox) *DiscoveredDeviceRepository {
	return &DiscoveredDeviceRepository{db: db, secretBox: secretBox}
}

func (r *DiscoveredDeviceRepository) List(ctx context.Context, filter interfaces.DiscoveredDeviceFilter) ([]domain.DiscoveredDevice, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 1)
	args := make([]any, 0, 4)
	next := 1

	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", next))
		args = append(args, string(filter.Status))
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM discovered_devices" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
SELECT
	device_id::text,
	ip_address,
	sys_object_id,
	detected_vendor,
	snmp_version,
	snmp_community,
	status,
	COALESCE(promoted_tower_id::text, ''),
	first_seen_at,
	last_seen_at,
	updated_at,
	created_at
FROM discovered_devices` + where + fmt.Sprintf(" ORDER BY first_seen_at DESC LIMIT $%d OFFSET $%d", next, next+1)

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	devices := make([]domain.DiscoveredDevice, 0)
	for rows.Next() {
		var d domain.DiscoveredDevice
		var status string
		if err := rows.Scan(
			&d.ID,
			&d.IPAddress,
			&d.SysObjectID,
			&d.DetectedVendor,
			&d.SNMPVersion,
			&d.SNMPCommunity,
			&status,
			&d.PromotedTowerID,
			&d.FirstSeenAt,
			&d.LastSeenAt,
			&d.UpdatedAt,
			&d.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		d.Status = domain.DiscoveredDeviceStatus(status)
		if err := r.decryptSecret(&d); err != nil {
			return nil, 0, err
		}
		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return devices, total, nil
}

func (r *DiscoveredDeviceRepository) GetByID(ctx context.Context, id string) (*domain.DiscoveredDevice, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	device_id::text,
	ip_address,
	sys_object_id,
	detected_vendor,
	snmp_version,
	snmp_community,
	status,
	COALESCE(promoted_tower_id::text, ''),
	first_seen_at,
	last_seen_at,
	updated_at,
	created_at
FROM discovered_devices
WHERE device_id::text = $1`

	var d domain.DiscoveredDevice
	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.IPAddress,
		&d.SysObjectID,
		&d.DetectedVendor,
		&d.SNMPVersion,
		&d.SNMPCommunity,
		&status,
		&d.PromotedTowerID,
		&d.FirstSeenAt,
		&d.LastSeenAt,
		&d.UpdatedAt,
		&d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrDiscoveredDeviceNotFound
		}
		return nil, err
	}
	d.Status = domain.DiscoveredDeviceStatus(status)
	if err := r.decryptSecret(&d); err != nil {
		return nil, err
	}
	return &d, nil
}

// UpsertSeen insere um novo dispositivo, ou atualiza last_seen_at e os
// campos detetados de um já existente — sem nunca sobrescrever Status,
// para não reabrir um dispositivo já promovido ou ignorado.
func (r *DiscoveredDeviceRepository) UpsertSeen(ctx context.Context, device *domain.DiscoveredDevice) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	encCommunity, err := r.encryptSecret(device.SNMPCommunity)
	if err != nil {
		return err
	}

	const query = `
INSERT INTO discovered_devices (
	ip_address, sys_object_id, detected_vendor, snmp_version, snmp_community, status, first_seen_at, last_seen_at, updated_at, created_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $8, $7
)
ON CONFLICT (ip_address) DO UPDATE SET
	sys_object_id = EXCLUDED.sys_object_id,
	detected_vendor = EXCLUDED.detected_vendor,
	snmp_version = EXCLUDED.snmp_version,
	snmp_community = EXCLUDED.snmp_community,
	last_seen_at = EXCLUDED.last_seen_at,
	updated_at = EXCLUDED.last_seen_at`

	_, err = r.db.ExecContext(
		ctx,
		query,
		device.IPAddress,
		device.SysObjectID,
		device.DetectedVendor,
		device.SNMPVersion,
		encCommunity,
		string(domain.DiscoveredDeviceStatusPending),
		device.FirstSeenAt,
		device.LastSeenAt,
	)
	return err
}

// UpdateStatus marca um dispositivo como promovido ou ignorado.
func (r *DiscoveredDeviceRepository) UpdateStatus(ctx context.Context, id string, status domain.DiscoveredDeviceStatus, promotedTowerID string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
UPDATE discovered_devices
SET status = $2, promoted_tower_id = NULLIF($3, '')::uuid, updated_at = NOW()
WHERE device_id::text = $1`

	result, err := r.db.ExecContext(ctx, query, id, string(status), promotedTowerID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return interfaces.ErrDiscoveredDeviceNotFound
	}
	return nil
}

func (r *DiscoveredDeviceRepository) encryptSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if r.secretBox == nil {
		return value, nil
	}
	return r.secretBox.Encrypt(value)
}

func (r *DiscoveredDeviceRepository) decryptSecret(d *domain.DiscoveredDevice) error {
	if r.secretBox == nil {
		return nil
	}
	var err error
	d.SNMPCommunity, err = r.secretBox.Decrypt(d.SNMPCommunity)
	return err
}
