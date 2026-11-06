package persistence

import (
	"context"
	"database/sql"
	"time"

	"eventdispatcher/internal/domain"
	"eventdispatcher/pkg/id"
)

type SubscriberRepository struct {
	db *DB
}

func NewSubscriberRepository(db *DB) *SubscriberRepository {
	return &SubscriberRepository{db: db}
}

func (r *SubscriberRepository) Create(ctx context.Context, sub *domain.Subscriber) error {
	if sub.ID == "" {
		newID, err := id.New("sub_")
		if err != nil {
			return err
		}
		sub.ID = newID
	}
	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	if sub.UpdatedAt.IsZero() {
		sub.UpdatedAt = now
	}

	query := `INSERT INTO subscribers (id, account_id, name, callback_url, secret, enabled, timeout_ms, max_attempts, filter_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.raw.ExecContext(ctx, query,
		sub.ID, sub.AccountID, sub.Name, sub.CallbackURL, sub.Secret,
		boolToInt(sub.Enabled), sub.TimeoutMs, sub.MaxAttempts, sub.FilterJSON,
		formatTime(sub.CreatedAt), formatTime(sub.UpdatedAt),
	)
	return mapDBError(err)
}

func (r *SubscriberRepository) GetByID(ctx context.Context, id string) (*domain.Subscriber, error) {
	query := `SELECT id, account_id, name, callback_url, secret, enabled, timeout_ms, max_attempts, filter_json, created_at, updated_at
		FROM subscribers WHERE id = ?`
	return r.scan(r.db.raw.QueryRowContext(ctx, query, id))
}

func (r *SubscriberRepository) ListByAccount(ctx context.Context, accountID string) ([]*domain.Subscriber, error) {
	var query string
	var args []any
	if accountID != "" {
		query = `SELECT id, account_id, name, callback_url, secret, enabled, timeout_ms, max_attempts, filter_json, created_at, updated_at
			FROM subscribers WHERE account_id = ? ORDER BY created_at ASC`
		args = append(args, accountID)
	} else {
		query = `SELECT id, account_id, name, callback_url, secret, enabled, timeout_ms, max_attempts, filter_json, created_at, updated_at
			FROM subscribers ORDER BY created_at ASC`
	}

	rows, err := r.db.raw.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var list []*domain.Subscriber
	for rows.Next() {
		sub, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, rows.Err()
}

func (r *SubscriberRepository) ListEnabledByAccount(ctx context.Context, accountID string) ([]*domain.Subscriber, error) {
	query := `SELECT id, account_id, name, callback_url, secret, enabled, timeout_ms, max_attempts, filter_json, created_at, updated_at
		FROM subscribers WHERE account_id = ? AND enabled = 1 ORDER BY created_at ASC`
	rows, err := r.db.raw.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var list []*domain.Subscriber
	for rows.Next() {
		sub, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, rows.Err()
}

func (r *SubscriberRepository) Update(ctx context.Context, sub *domain.Subscriber) error {
	sub.UpdatedAt = time.Now().UTC()
	query := `UPDATE subscribers SET account_id = ?, name = ?, callback_url = ?, secret = ?, enabled = ?, timeout_ms = ?, max_attempts = ?, filter_json = ?, updated_at = ?
		WHERE id = ?`
	res, err := r.db.raw.ExecContext(ctx, query,
		sub.AccountID, sub.Name, sub.CallbackURL, sub.Secret,
		boolToInt(sub.Enabled), sub.TimeoutMs, sub.MaxAttempts, sub.FilterJSON,
		formatTime(sub.UpdatedAt), sub.ID,
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

func (r *SubscriberRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM subscribers WHERE id = ?`
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

func (r *SubscriberRepository) scan(s rowScanner) (*domain.Subscriber, error) {
	var sub domain.Subscriber
	var enabledInt int
	var cTime, uTime string
	err := s.Scan(
		&sub.ID, &sub.AccountID, &sub.Name, &sub.CallbackURL, &sub.Secret,
		&enabledInt, &sub.TimeoutMs, &sub.MaxAttempts, &sub.FilterJSON,
		&cTime, &uTime,
	)
	if err != nil {
		return nil, mapDBError(err)
	}
	sub.Enabled = enabledInt == 1
	sub.CreatedAt = parseTime(cTime)
	sub.UpdatedAt = parseTime(uTime)
	return &sub, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
