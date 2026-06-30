package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type ExecutionRecord struct {
	ID              string
	ScriptName      string
	Branch          string
	Environment     string
	ExitCode        int
	DurationSeconds float64
	ExecutedAt      time.Time
	Error           string
}

type Client struct {
	db *sql.DB
}

func NewClient(connStr string) (*Client, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{db: db}, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS execution_records (
		id VARCHAR(36) PRIMARY KEY,
		script_name VARCHAR(255) NOT NULL,
		branch VARCHAR(255) NOT NULL,
		environment VARCHAR(50) NOT NULL,
		exit_code INT NOT NULL,
		duration_seconds DOUBLE PRECISION NOT NULL,
		executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		error TEXT
	);
	`
	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute schema creation: %w", err)
	}
	return nil
}

func (c *Client) InsertExecution(ctx context.Context, record ExecutionRecord) error {
	query := `
	INSERT INTO execution_records (id, script_name, branch, environment, exit_code, duration_seconds, error)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var errStr *string
	if record.Error != "" {
		errStr = &record.Error
	}
	_, err := c.db.ExecContext(ctx, query,
		record.ID,
		record.ScriptName,
		record.Branch,
		record.Environment,
		record.ExitCode,
		record.DurationSeconds,
		errStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert execution record: %w", err)
	}
	return nil
}

func (c *Client) PurgeRecords(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `
	DELETE FROM execution_records
	WHERE executed_at < $1
	`
	res, err := c.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to purge records: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	return rowsAffected, nil
}
