package persistence

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"eventdispatcher/internal/domain"
)

type DeliveryRepository struct {
	db *DB
}

func NewDeliveryRepository(db *DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

func (r *DeliveryRepository) ClaimDue(ctx context.Context, now time.Time, limit int) ([]domain.DueJob, error) {
	if limit <= 0 {
		limit = 20
	}
	tx, err := r.db.raw.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	querySelect := `SELECT d.id, d.event_id, d.subscriber_id, s.callback_url, s.secret, s.timeout_ms, s.max_attempts, d.attempt, e.payload_json
		FROM deliveries d
		JOIN subscribers s ON d.subscriber_id = s.id
		JOIN events e ON d.event_id = e.id
		WHERE d.status = ? AND d.next_retry_at <= ? AND s.enabled = 1
		ORDER BY d.next_retry_at ASC LIMIT ?`

	rows, err := tx.QueryContext(ctx, querySelect, domain.DeliveryStatusPending, now.Unix(), limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var jobs []domain.DueJob
	var ids []any
	for rows.Next() {
		var j domain.DueJob
		var timeoutMs int
		var payloadStr string
		err := rows.Scan(
			&j.ID, &j.EventID, &j.SubscriberID, &j.CallbackURL, &j.Secret,
			&timeoutMs, &j.MaxAttempts, &j.Attempt, &payloadStr,
		)
		if err != nil {
			return nil, err
		}
		j.Timeout = time.Duration(timeoutMs) * time.Millisecond
		j.Payload = []byte(payloadStr)
		jobs = append(jobs, j)
		ids = append(ids, j.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(jobs) == 0 {
		return nil, tx.Commit()
	}

	queryUpdate := `UPDATE deliveries SET status = ?, attempt = attempt + 1, updated_at = ? WHERE id IN (` + placeholders(len(ids)) + `)`
	args := []any{domain.DeliveryStatusInflight, formatTime(now)}
	args = append(args, ids...)
	if _, err := tx.ExecContext(ctx, queryUpdate, args...); err != nil {
		return nil, err
	}

	for i := range jobs {
		jobs[i].Attempt++
	}

	return jobs, tx.Commit()
}

func (r *DeliveryRepository) Settle(ctx context.Context, id, status string, statusCode int, errMsg string, nextRetryAt, updatedAt time.Time) error {
	query := `UPDATE deliveries
		SET status = ?, last_status_code = ?, last_error = ?, next_retry_at = ?, updated_at = ?
		WHERE id = ?`
	_, err := r.db.raw.ExecContext(ctx, query, status, statusCode, errMsg, nextRetryAt.Unix(), formatTime(updatedAt), id)
	return mapDBError(err)
}

func (r *DeliveryRepository) RequeueInflight(ctx context.Context, now time.Time) (int64, error) {
	query := `UPDATE deliveries
		SET status = ?, next_retry_at = ?, updated_at = ?
		WHERE status = ?`
	res, err := r.db.raw.ExecContext(ctx, query, domain.DeliveryStatusPending, now.Unix(), formatTime(now), domain.DeliveryStatusInflight)
	if err != nil {
		return 0, mapDBError(err)
	}
	return res.RowsAffected()
}

func (r *DeliveryRepository) RequeueUnsent(ctx context.Context, id string, now time.Time) error {
	query := `UPDATE deliveries
		SET status = ?, attempt = MAX(0, attempt - 1), next_retry_at = ?, updated_at = ?
		WHERE id = ? AND status = ?`
	_, err := r.db.raw.ExecContext(ctx, query, domain.DeliveryStatusPending, now.Unix(), formatTime(now), id, domain.DeliveryStatusInflight)
	return mapDBError(err)
}

func (r *DeliveryRepository) Reset(ctx context.Context, id string, now time.Time) error {
	query := `UPDATE deliveries
		SET status = ?, attempt = 0, next_retry_at = ?, last_status_code = 0, last_error = '', updated_at = ?
		WHERE id = ?`
	res, err := r.db.raw.ExecContext(ctx, query, domain.DeliveryStatusPending, now.Unix(), formatTime(now), id)
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

func (r *DeliveryRepository) GetByID(ctx context.Context, id string) (*domain.Delivery, error) {
	query := `SELECT d.id, d.event_id, d.subscriber_id, s.name, s.callback_url, d.status, d.attempt, d.next_retry_at, d.last_status_code, d.last_error, d.updated_at
		FROM deliveries d
		JOIN subscribers s ON d.subscriber_id = s.id
		WHERE d.id = ?`
	return r.scan(r.db.raw.QueryRowContext(ctx, query, id))
}

func (r *DeliveryRepository) List(ctx context.Context, f domain.DeliveryFilter, page, limit int) ([]*domain.Delivery, int, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 15
	}
	offset := (page - 1) * limit

	var where []string
	var args []any
	if f.EventID != "" {
		where = append(where, "d.event_id = ?")
		args = append(args, f.EventID)
	}
	if f.SubscriberID != "" {
		where = append(where, "d.subscriber_id = ?")
		args = append(args, f.SubscriberID)
	}
	if f.Status != "" {
		where = append(where, "d.status = ?")
		args = append(args, f.Status)
	}
	if f.DeliveryID != "" {
		where = append(where, "d.id = ?")
		args = append(args, f.DeliveryID)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " AND " + strings.Join(where, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM deliveries d JOIN subscribers s ON d.subscriber_id = s.id WHERE 1=1" + whereClause
	var total int
	if err := r.db.raw.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, mapDBError(err)
	}
	if total == 0 {
		return []*domain.Delivery{}, 0, nil
	}

	query := `SELECT d.id, d.event_id, d.subscriber_id, s.name, s.callback_url, d.status, d.attempt, d.next_retry_at, d.last_status_code, d.last_error, d.updated_at
		FROM deliveries d
		JOIN subscribers s ON d.subscriber_id = s.id WHERE 1=1` + whereClause + " ORDER BY d.updated_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	rows, err := r.db.raw.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, mapDBError(err)
	}
	defer rows.Close()

	var list []*domain.Delivery
	for rows.Next() {
		item, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, item)
	}
	return list, total, rows.Err()
}

func (r *DeliveryRepository) scan(s rowScanner) (*domain.Delivery, error) {
	var d domain.Delivery
	var nextRetryUnix int64
	var uTime string
	err := s.Scan(
		&d.ID, &d.EventID, &d.SubscriberID, &d.SubscriberName, &d.CallbackURL, &d.Status,
		&d.Attempt, &nextRetryUnix, &d.LastStatusCode, &d.LastError, &uTime,
	)
	if err != nil {
		return nil, mapDBError(err)
	}
	d.NextRetryAt = time.Unix(nextRetryUnix, 0).UTC()
	d.UpdatedAt = parseTime(uTime)
	return &d, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n*2-1)
	for i := 0; i < n; i++ {
		b[i*2] = '?'
		if i < n-1 {
			b[i*2+1] = ','
		}
	}
	return string(b)
}
