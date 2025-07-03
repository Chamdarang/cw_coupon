package store

import (
	"context"
	"database/sql"
	"fmt"
)

type DBStore struct {
	db *sql.DB
}

func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// Campaign model
type Campaign struct {
	ID             string
	Name           string
	StartTimestamp int64
	TotalCoupons   int
}

func (d *DBStore) CreateCampaign(ctx context.Context, id string, name string, startTimestamp int64, totalCoupons int) error {
	_, err := d.db.ExecContext(ctx, `
INSERT INTO campaigns (id, name, start_time, total_coupons)
VALUES (?, ?, FROM_UNIXTIME(?), ?)
`, id, name, startTimestamp, totalCoupons)
	return err
}

func (d *DBStore) GetCampaign(ctx context.Context, id string) (*Campaign, error) {
	row := d.db.QueryRowContext(ctx, `
SELECT id, name, UNIX_TIMESTAMP(start_time), total_coupons
FROM campaigns
WHERE id = ?
`, id)

	var c Campaign
	if err := row.Scan(&c.ID, &c.Name, &c.StartTimestamp, &c.TotalCoupons); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("campaign not found")
		}
		return nil, err
	}
	return &c, nil
}

func (d *DBStore) SaveCoupon(ctx context.Context, campaignID, code string) error {
	_, err := d.db.ExecContext(ctx, `
INSERT INTO coupons (campaign_id, coupon_code, issued_at)
VALUES (?, ?, NOW())
`, campaignID, code)
	return err
}

func (d *DBStore) GetCampaignWithCoupons(ctx context.Context, id string) (*Campaign, []string, error) {
	campaign, err := d.GetCampaign(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	rows, err := d.db.QueryContext(ctx, `
SELECT coupon_code FROM coupons WHERE campaign_id = ?
`, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, nil, err
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return campaign, codes, nil
}
