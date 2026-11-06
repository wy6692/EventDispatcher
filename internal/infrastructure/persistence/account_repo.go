package persistence

import (
	"context"
	"database/sql"
	"time"

	"eventdispatcher/internal/domain"
	"eventdispatcher/pkg/id"
)

type AccountRepository struct {
	db *DB
}

func NewAccountRepository(db *DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, acc *domain.Account) error {
	if acc.ID == "" {
		newID, err := id.New("acc_")
		if err != nil {
			return err
		}
		acc.ID = newID
	}
	now := time.Now().UTC()
	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = now
	}
	if acc.UpdatedAt.IsZero() {
		acc.UpdatedAt = now
	}

	query := `INSERT INTO accounts (id, name, account_key, app_id, token, encoding_aes_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.raw.ExecContext(ctx, query,
		acc.ID, acc.Name, acc.AccountKey, acc.AppID, acc.Token, acc.EncodingAESKey,
		formatTime(acc.CreatedAt), formatTime(acc.UpdatedAt),
	)
	return mapDBError(err)
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	query := `SELECT id, name, account_key, app_id, token, encoding_aes_key, created_at, updated_at
		FROM accounts WHERE id = ?`
	return r.scanOne(r.db.raw.QueryRowContext(ctx, query, id))
}

func (r *AccountRepository) GetByKey(ctx context.Context, accountKey string) (*domain.Account, error) {
	query := `SELECT id, name, account_key, app_id, token, encoding_aes_key, created_at, updated_at
		FROM accounts WHERE account_key = ?`
	return r.scanOne(r.db.raw.QueryRowContext(ctx, query, accountKey))
}

func (r *AccountRepository) List(ctx context.Context) ([]*domain.Account, error) {
	query := `SELECT id, name, account_key, app_id, token, encoding_aes_key, created_at, updated_at
		FROM accounts ORDER BY created_at ASC`
	rows, err := r.db.raw.QueryContext(ctx, query)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var list []*domain.Account
	for rows.Next() {
		acc, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, acc)
	}
	return list, rows.Err()
}

func (r *AccountRepository) Update(ctx context.Context, acc *domain.Account) error {
	acc.UpdatedAt = time.Now().UTC()
	query := `UPDATE accounts SET name = ?, account_key = ?, app_id = ?, token = ?, encoding_aes_key = ?, updated_at = ?
		WHERE id = ?`
	res, err := r.db.raw.ExecContext(ctx, query,
		acc.Name, acc.AccountKey, acc.AppID, acc.Token, acc.EncodingAESKey, formatTime(acc.UpdatedAt), acc.ID,
	)
	if err != nil {
		return mapDBError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM accounts WHERE id = ?`
	res, err := r.db.raw.ExecContext(ctx, query, id)
	if err != nil {
		return mapDBError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *AccountRepository) scanOne(row *sql.Row) (*domain.Account, error) {
	var a domain.Account
	var cTime, uTime string
	err := row.Scan(&a.ID, &a.Name, &a.AccountKey, &a.AppID, &a.Token, &a.EncodingAESKey, &cTime, &uTime)
	if err != nil {
		return nil, mapDBError(err)
	}
	a.CreatedAt = parseTime(cTime)
	a.UpdatedAt = parseTime(uTime)
	return &a, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *AccountRepository) scan(s rowScanner) (*domain.Account, error) {
	var a domain.Account
	var cTime, uTime string
	err := s.Scan(&a.ID, &a.Name, &a.AccountKey, &a.AppID, &a.Token, &a.EncodingAESKey, &cTime, &uTime)
	if err != nil {
		return nil, mapDBError(err)
	}
	a.CreatedAt = parseTime(cTime)
	a.UpdatedAt = parseTime(uTime)
	return &a, nil
}
