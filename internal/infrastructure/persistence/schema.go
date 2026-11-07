package persistence

var SchemaDDL = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		account_key TEXT NOT NULL UNIQUE,
		app_id TEXT NOT NULL,
		token TEXT NOT NULL,
		encoding_aes_key TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS subscribers (
		id TEXT PRIMARY KEY,
		account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		callback_url TEXT NOT NULL,
		secret TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		timeout_ms INTEGER NOT NULL,
		max_attempts INTEGER NOT NULL,
		filter_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_subscribers_account ON subscribers(account_id)`,
	`CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		dedupe_key TEXT NOT NULL UNIQUE,
		msg_type TEXT NOT NULL,
		event TEXT NOT NULL DEFAULT '',
		event_key TEXT NOT NULL DEFAULT '',
		from_user TEXT NOT NULL DEFAULT '',
		to_user TEXT NOT NULL DEFAULT '',
		create_time INTEGER NOT NULL DEFAULT 0,
		payload_json TEXT NOT NULL,
		received_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_events_account_received ON events(account_id, received_at)`,
	`CREATE INDEX IF NOT EXISTS idx_events_received_at ON events(received_at)`,
	`CREATE TABLE IF NOT EXISTS deliveries (
		id TEXT PRIMARY KEY,
		event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
		subscriber_id TEXT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		attempt INTEGER NOT NULL DEFAULT 0,
		next_retry_at INTEGER NOT NULL,
		last_status_code INTEGER NOT NULL DEFAULT 0,
		last_error TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL,
		UNIQUE(event_id, subscriber_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_deliveries_due ON deliveries(status, next_retry_at)`,
}
