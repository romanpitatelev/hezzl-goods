package clickhouse

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed migrations
var clickhouseMigrations embed.FS

type Store struct {
	db  *sql.DB
	dsn string
}

type Config struct {
	Dsn string
}

func New(ctx context.Context, cfg Config) (*Store, error) {
	opt, err := clickhouse.ParseDSN(cfg.Dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid ClickHouse DSN: %w", err)
	}

	db := clickhouse.OpenDB(opt)
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &Store{
		db:  db,
		dsn: cfg.Dsn,
	}, nil
}

func (c *Store) Migrate(direction migrate.MigrationDirection) error {
	suffix := "up.sql"
	if direction == migrate.Down {
		suffix = "down.sql"
	}

	entries, err := clickhouseMigrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}

		content, err := fs.ReadFile(clickhouseMigrations, filepath.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if _, err := c.db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// func (c *Store) Begin() (*sql.Tx, error) {
// 	tx, err := c.db.Begin()
// 	if err != nil {
// 		return nil, fmt.Errorf("error starting ClickHouse: %w", err)
// 	}

// 	return tx, nil
// }

func (c *Store) Close() error {
	if err := c.db.Close(); err != nil {
		return fmt.Errorf("error closing ClickHouse: %w", err)
	}

	return nil
}

func (c *Store) Truncate(ctx context.Context, tables ...string) error {
	for _, table := range tables {
		if _, err := c.db.ExecContext(ctx, fmt.Sprintf(`
			CREATE TABLE %s_tmp AS %s
			ENGINE = MergeTree()
			ORDER BY (event_time, id)
			PARTITION BY (event_time, id, project_id)
			SETTINGS index_granularity = 8192	
		`, table, table)); err != nil {
			return fmt.Errorf("error creating temp table for %s: %w", table, err)
		}

		if _, err := c.db.ExecContext(ctx, fmt.Sprintf(`
			RENAME TABLE %s TO %s_old, %s_tmp TO %s
		`, table, table, table, table)); err != nil {
			return fmt.Errorf("error renaming tables for %s: %w", table, err)
		}

		if _, err := c.db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE %s_old`, table)); err != nil {
			return fmt.Errorf("error dropping old table %s: %w", table, err)
		}
	}

	return nil
}

func (c *Store) DB() *sql.DB {
	return c.db
}
