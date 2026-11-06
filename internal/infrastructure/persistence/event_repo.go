package persistence

import (
	"context"
	"strings"
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

func (r *EventRepository) List(ctx context.Context, f domain.EventFilter, page, limit int) ([]*domain.Event, int, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 15
	}
	offset := (page - 1) * limit

	var where []string
	var args []any
	if f.AccountID != "" {
		where = append(where, "account_id = ?")
		args = append(args, f.AccountID)
	}
	if f.MsgType != "" {
		where = append(where, "msg_type = ?")
		args = append(args, f.MsgType)
	}
	if f.Event != "" {
		where = append(where, "event = ?")
		args = append(args, f.Event)
	}
	if f.FromUser != "" {
		where = append(where, "from_user = ?")
		args = append(args, f.FromUser)
	}
	if f.EventID != "" {
		where = append(where, "id = ?")
		args = append(args, f.EventID)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM events" + whereClause
	var total int
	if err := r.db.raw.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, mapDBError(err)
	}
	if total == 0 {
		return []*domain.Event{}, 0, nil
	}

	query := "SELECT id, account_id, dedupe_key, msg_type, event, event_key, from_user, to_user, create_time, payload_json, received_at FROM events" +
		whereClause + " ORDER BY received_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	rows, err := r.db.raw.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, mapDBError(err)
	}
	defer rows.Close()

	var list []*domain.Event
	for rows.Next() {
		evt, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, evt)
	}
	return list, total, rows.Err()
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
