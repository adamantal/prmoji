package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlCreateTablePRMessages = `CREATE TABLE IF NOT EXISTS pr_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		inserted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		pr_url TEXT NOT NULL,
		message_channel TEXT,
		message_timestamp TEXT
	);`

	sqlCreateIndexPRMessagesPRURL = `CREATE INDEX IF NOT EXISTS idx_pr_messages_pr_url ON pr_messages(pr_url);`

	sqlCreateIndexPRMessagesInsertedAt = `CREATE INDEX IF NOT EXISTS idx_pr_messages_inserted_at ON pr_messages(inserted_at);`

	sqlInsertPRMessage = `INSERT INTO pr_messages(pr_url, message_channel, message_timestamp) VALUES(?, ?, ?);`

	sqlSelectMessagesByPRURL = `SELECT id, inserted_at, pr_url, message_channel, message_timestamp FROM pr_messages WHERE pr_url = ?;`

	sqlDeleteMessagesByPRURL = `DELETE FROM pr_messages WHERE pr_url = ?;`

	sqlDeleteMessagesOlderThanDate = `DELETE FROM pr_messages WHERE date(inserted_at) < date(?);`
)

type Message struct {
	ID               int64
	InsertedAt       time.Time
	PRURL            string
	MessageChannel   string
	MessageTimestamp string
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite is easy to lock under concurrent writes; keep it simple.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	s := &SQLiteStore{db: db}
	if err := s.initSchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) initSchema(ctx context.Context) error {
	stmts := []string{
		sqlCreateTablePRMessages,
		sqlCreateIndexPRMessagesPRURL,
		sqlCreateIndexPRMessagesInsertedAt,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	slog.Info("sqlite schema initialized")
	return nil
}

func (s *SQLiteStore) InsertPRMessage(ctx context.Context, prURL, channel, ts string) error {
	_, err := s.db.ExecContext(
		ctx,
		sqlInsertPRMessage,
		prURL,
		channel,
		ts,
	)
	if err != nil {
		return fmt.Errorf("insert pr message: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListMessagesByPRURL(ctx context.Context, prURL string) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		sqlSelectMessagesByPRURL,
		prURL,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.InsertedAt, &m.PRURL, &m.MessageChannel, &m.MessageTimestamp); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) DeleteByPRURL(ctx context.Context, prURL string) error {
	_, err := s.db.ExecContext(ctx, sqlDeleteMessagesByPRURL, prURL)
	if err != nil {
		return fmt.Errorf("delete by pr_url: %w", err)
	}
	return nil
}

// DeleteOlderThanDate deletes rows whose inserted_at date is strictly older than cutoffDate (date-only compare).
func (s *SQLiteStore) DeleteOlderThanDate(ctx context.Context, cutoffDate time.Time) (int64, error) {
	cutoff := cutoffDate.UTC().Format("2006-01-02")
	res, err := s.db.ExecContext(ctx, sqlDeleteMessagesOlderThanDate, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete older than: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
