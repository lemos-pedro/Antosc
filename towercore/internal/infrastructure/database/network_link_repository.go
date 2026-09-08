package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type networkLinkRepository struct{ db *sql.DB }

func NewNetworkLinkRepository(db *sql.DB) interfaces.NetworkLinkRepository {
	return &networkLinkRepository{db: db}
}

func (r *networkLinkRepository) Create(ctx context.Context, link *domain.NetworkLink) error {
	if link.LinkID == "" {
		link.LinkID = uuid.NewString()
	}
	now := time.Now().UTC()
	if link.CreatedAt.IsZero() {
		link.CreatedAt = now
	}
	if link.UpdatedAt.IsZero() {
		link.UpdatedAt = now
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO network_links
 (link_id,name,router_host,router_ip,if_index,if_descr,if_alias,operator,media_type,nominal_capacity_mb,region_id,created_at,updated_at)
 VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid,$12,$13)`, link.LinkID, link.Name, link.RouterHost, link.RouterIP, link.IfIndex, link.IfDescr, link.IfAlias, link.Operator, link.MediaType, link.NominalCapacityMb, link.RegionID, link.CreatedAt, link.UpdatedAt)
	return err
}

func (r *networkLinkRepository) Update(ctx context.Context, link *domain.NetworkLink) error {
	link.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `UPDATE network_links SET name=$2,router_host=$3,router_ip=$4,if_index=$5,if_descr=$6,if_alias=$7,operator=$8,media_type=$9,nominal_capacity_mb=$10,region_id=NULLIF($11,'')::uuid,updated_at=$12 WHERE link_id=$1::uuid`, link.LinkID, link.Name, link.RouterHost, link.RouterIP, link.IfIndex, link.IfDescr, link.IfAlias, link.Operator, link.MediaType, link.NominalCapacityMb, link.RegionID, link.UpdatedAt)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

const networkLinkColumns = `link_id::text,name,router_host,router_ip,if_index,COALESCE(if_descr,''),COALESCE(if_alias,''),COALESCE(operator,''),media_type,nominal_capacity_mb,COALESCE(region_id::text,''),created_at,updated_at`

func scanNetworkLink(scanner interface{ Scan(...any) error }) (*domain.NetworkLink, error) {
	link := &domain.NetworkLink{}
	err := scanner.Scan(&link.LinkID, &link.Name, &link.RouterHost, &link.RouterIP, &link.IfIndex, &link.IfDescr, &link.IfAlias, &link.Operator, &link.MediaType, &link.NominalCapacityMb, &link.RegionID, &link.CreatedAt, &link.UpdatedAt)
	return link, err
}

func (r *networkLinkRepository) GetByID(ctx context.Context, id string) (*domain.NetworkLink, error) {
	link, err := scanNetworkLink(r.db.QueryRowContext(ctx, `SELECT `+networkLinkColumns+` FROM network_links WHERE link_id=$1::uuid`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (r *networkLinkRepository) GetByRouterAndIfIndex(ctx context.Context, routerIP string, ifIndex int) (*domain.NetworkLink, error) {
	link, err := scanNetworkLink(r.db.QueryRowContext(ctx, `SELECT `+networkLinkColumns+` FROM network_links WHERE router_ip=$1 AND if_index=$2`, routerIP, ifIndex))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return link, nil
}

func (r *networkLinkRepository) List(ctx context.Context, limit, offset int) ([]*domain.NetworkLink, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+networkLinkColumns+` FROM network_links ORDER BY name,router_ip,if_index LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	links := make([]*domain.NetworkLink, 0)
	for rows.Next() {
		link, e := scanNetworkLink(rows)
		if e != nil {
			return nil, 0, e
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_links`).Scan(&total); err != nil {
		return nil, 0, err
	}
	return links, total, nil
}

func (r *networkLinkRepository) SaveMetricSnapshot(ctx context.Context, s *domain.LinkMetricSnapshot) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO link_metric_snapshots (link_id,collected_at,oper_status,admin_status,in_octets,out_octets,in_errors,out_errors,in_discards,out_discards,speed_mb,in_util_percent,out_util_percent) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, s.LinkID, s.CollectedAt, string(s.OperStatus), string(s.AdminStatus), int64(s.InOctets), int64(s.OutOctets), int64(s.InErrors), int64(s.OutErrors), int64(s.InDiscards), int64(s.OutDiscards), s.SpeedMb, s.InUtilPercent, s.OutUtilPercent)
	return err
}

func (r *networkLinkRepository) GetLastSnapshot(ctx context.Context, linkID string) (*domain.LinkMetricSnapshot, error) {
	var s domain.LinkMetricSnapshot
	var inOct, outOct, inErr, outErr, inDis, outDis int64
	err := r.db.QueryRowContext(ctx, `SELECT collected_at,oper_status,admin_status,in_octets,out_octets,in_errors,out_errors,in_discards,out_discards,speed_mb,in_util_percent,out_util_percent FROM link_metric_snapshots WHERE link_id=$1::uuid ORDER BY collected_at DESC LIMIT 1`, linkID).Scan(&s.CollectedAt, &s.OperStatus, &s.AdminStatus, &inOct, &outOct, &inErr, &outErr, &inDis, &outDis, &s.SpeedMb, &s.InUtilPercent, &s.OutUtilPercent)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get last link snapshot: %w", err)
	}
	s.LinkID = linkID
	s.InOctets = uint64(maxInt64(inOct))
	s.OutOctets = uint64(maxInt64(outOct))
	s.InErrors = uint64(maxInt64(inErr))
	s.OutErrors = uint64(maxInt64(outErr))
	s.InDiscards = uint64(maxInt64(inDis))
	s.OutDiscards = uint64(maxInt64(outDis))
	return &s, nil
}

func maxInt64(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}
