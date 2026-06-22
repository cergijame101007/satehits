package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresSupplierRepository はPostgreSQLを使った取引先リポジトリの実装
type PostgresSupplierRepository struct {
	baseRepository
}

// NewPostgresSupplierRepository はPostgresSupplierRepositoryのインスタンスを作成する
func NewPostgresSupplierRepository(db *sql.DB) *PostgresSupplierRepository {
	return &PostgresSupplierRepository{baseRepository{db: db}}
}

const supplierSelectColumns = `
SELECT id, name, description, instagram_url, image_url, display_order, is_active, created_at, updated_at
FROM suppliers`

func scanSupplier(row interface {
	Scan(dest ...any) error
}) (domain.Supplier, error) {
	var (
		s            domain.Supplier
		instagramURL sql.NullString
		imageURL     sql.NullString
	)
	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.Description,
		&instagramURL,
		&imageURL,
		&s.DisplayOrder,
		&s.IsActive,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return domain.Supplier{}, err
	}
	if instagramURL.Valid {
		s.InstagramURL = &instagramURL.String
	}
	if imageURL.Valid {
		s.ImageURL = &imageURL.String
	}
	return s, nil
}

// List は取引先一覧を display_order ASC, id ASC で返す。
// activeOnly が true の場合は is_active=true のみを返す（公開用フィルタはここで固定する）。
func (r *PostgresSupplierRepository) List(ctx context.Context, activeOnly bool) ([]domain.Supplier, error) {
	query := supplierSelectColumns
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY display_order ASC, id ASC"

	rows, err := r.getDB(ctx).QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []domain.Supplier
	for rows.Next() {
		s, err := scanSupplier(rows)
		if err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suppliers, nil
}

// GetByID は ID で取引先を1件取得する
func (r *PostgresSupplierRepository) GetByID(ctx context.Context, id int64) (domain.Supplier, error) {
	query := supplierSelectColumns + " WHERE id = $1"
	s, err := scanSupplier(r.getDB(ctx).QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Supplier{}, domain.ErrSupplierNotFound
		}
		return domain.Supplier{}, err
	}
	return s, nil
}

// Create は取引先を永続化し挿入結果を返す。
// DisplayOrder が nil の場合は末尾（COALESCE(MAX(display_order),0)+1）に採番する。
func (r *PostgresSupplierRepository) Create(ctx context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
	query := `
INSERT INTO suppliers (name, description, instagram_url, image_url, display_order, is_active)
VALUES ($1, $2, $3, $4, COALESCE($5, (SELECT COALESCE(MAX(display_order), 0) + 1 FROM suppliers)), $6)
RETURNING id, name, description, instagram_url, image_url, display_order, is_active, created_at, updated_at`

	var displayOrder sql.NullInt64
	if in.DisplayOrder != nil {
		displayOrder = sql.NullInt64{Int64: int64(*in.DisplayOrder), Valid: true}
	}

	s, err := scanSupplier(r.getDB(ctx).QueryRowContext(ctx, query,
		in.Name,
		in.Description,
		nullString(in.InstagramURL),
		nullString(in.ImageURL),
		displayOrder,
		in.IsActive,
	))
	if err != nil {
		return domain.Supplier{}, err
	}
	return s, nil
}

// Update は取引先の全項目を上書きする
func (r *PostgresSupplierRepository) Update(ctx context.Context, id int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
	query := `
UPDATE suppliers
SET name = $1, description = $2, instagram_url = $3, image_url = $4, display_order = $5, is_active = $6
WHERE id = $7
RETURNING id, name, description, instagram_url, image_url, display_order, is_active, created_at, updated_at`

	s, err := scanSupplier(r.getDB(ctx).QueryRowContext(ctx, query,
		in.Name,
		in.Description,
		nullString(in.InstagramURL),
		nullString(in.ImageURL),
		in.DisplayOrder,
		in.IsActive,
		id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Supplier{}, domain.ErrSupplierNotFound
		}
		return domain.Supplier{}, err
	}
	return s, nil
}

// Delete は取引先を削除する。対象が存在しない場合は ErrSupplierNotFound を返す。
func (r *PostgresSupplierRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.getDB(ctx).ExecContext(ctx, `DELETE FROM suppliers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrSupplierNotFound
	}
	return nil
}

// Reorder は ids の並び順に従って display_order を 1..N で更新する。
// 原子性のため呼び出し側でトランザクション（getDB が tx を返す）を張る前提。
func (r *PostgresSupplierRepository) Reorder(ctx context.Context, ids []int64) error {
	db := r.getDB(ctx)
	for i, id := range ids {
		res, err := db.ExecContext(ctx, `UPDATE suppliers SET display_order = $1 WHERE id = $2`, i+1, id)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrSupplierNotFound
		}
	}
	return nil
}

// UpdateImageURL は画像 URL のみを更新する
func (r *PostgresSupplierRepository) UpdateImageURL(ctx context.Context, id int64, imageURL string) (domain.Supplier, error) {
	query := `
UPDATE suppliers SET image_url = $1 WHERE id = $2
RETURNING id, name, description, instagram_url, image_url, display_order, is_active, created_at, updated_at`
	s, err := scanSupplier(r.getDB(ctx).QueryRowContext(ctx, query, imageURL, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Supplier{}, domain.ErrSupplierNotFound
		}
		return domain.Supplier{}, err
	}
	return s, nil
}

func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
