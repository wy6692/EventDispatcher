package persistence

import (
	"context"
	"time"

	"eventdispatcher/internal/domain"
	"eventdispatcher/pkg/id"
)

type EventRepository struct {
	db *DB
}

func NewEventRepository(db *DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) SaveIncoming(ctx context.Context, evt *domain.Event, subscriberIDs []string) (bool, error) {
	if evt.ID == "" {
		newID, err := id.New("evt_")
		if err != nil {
			return false, err
		}
		evt.ID = newID
	}
	if evt.ReceivedAt.IsZero() {
		evt.ReceivedAt = time.Now().UTC()
	}

	tx, err := r.db.raw.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	queryInsert := `INSERT INTO events (id, account_id, dedupe_key, msg_type, event, event_key, from_user, to_user, create_time, payload_json, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(dedupe_key) DO NOTHING`
	res, err := tx.ExecContext(ctx, queryInsert,
		evt.ID, evt.AccountID, evt.DedupeKey, evt.MsgType, evt.Event, evt.EventKey,
		evt.FromUser, evt.ToUser, evt.CreateTime, string(evt.PayloadJSON), formatTime(evt.ReceivedAt),
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, tx.Commit()
	}

	for _, subID := range subscriberIDs {
		dlvID, err := id.New("dlv_")
		if err != nil {
			return false, err
		}
		queryDlv := `INSERT INTO deliveries (id, event_id, subscriber_id, status, attempt, next_retry_at, last_status_code, last_error, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
		_, err = tx.ExecContext(ctx, queryDlv,
			dlvID, evt.ID, subID, domain.DeliveryStatusPending, 0, evt.ReceivedAt.Unix(), 0, "", formatTime(evt.ReceivedAt),
		)
		if err != nil {
			return false, err
		}
	}

	return true, tx.Commit()
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `SELECT id, account_id, dedupe_key, msg_type, event, event_key, from_user, to_user, create_time, payload_json, received_at
		FROM events WHERE id = ?`
	return r.scan(r.db.raw.QueryRowContext(ctx, query, id))
}

func (r *EventRepository) List(ctx context.Context, accountID string, limit int) ([]*domain.Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var query string
	var args []any
	if accountID != "" {
		query = `SELECT id, account_id, dedupe_key, msg_type, event, event_key, from_user, to_user, create_time, payload_json, received_at
			FROM events WHERE account_id = ? ORDER BY received_at DESC LIMIT ?`
		args = append(args, accountID, limit)
	} else {
		query = `SELECT id, account_id, dedupe_key, msg_type, event, event_key, from_user, to_user, create_time, payload_json, received_at
			FROM events ORDER BY received_at DESC LIMIT ?`
		args = append(args, limit)
	}

	rows, err := r.db.raw.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var list []*domain.Event
	for rows.Next() {
		evt, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, evt)
	}
	return list, rows.Err()
}

func (r *EventRepository) scan(s rowScanner) (*domain.Event, error) {
	var e domain.Event
	var payloadStr, rTime string
	err := s.Scan(
		&e.ID, &e.AccountID, &e.DedupeKey, &e.MsgType, &e.Event, &e.EventKey,
		&e.FromUser, &e.ToUser, &e.CreateTime, &payloadStr, &rTime,
	)
	if err != nil {
		return nil, mapDBError(err)
	}
	e.PayloadJSON = []byte(payloadStr)
	e.ReceivedAt = parseTime(rTime)
	return &e, nil
}
