package store

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
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

//go:embed clickhouse_migrations
var clickhouseMigrations embed.FS

type ClickHouseStore struct {
	db  *sql.DB
	dsn string
}

func NewClickHouse(ctx context.Context, dsn string) (*ClickHouseStore, error) {
	opt, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid ClickHouse DSN: %w", err)
	}

	log.Debug().Msgf("dsn in NewClickHouse() function: %v", dsn)

	db := clickhouse.OpenDB(opt)
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &ClickHouseStore{
		db:  db,
		dsn: dsn,
	}, nil
}

func (c *ClickHouseStore) Migrate(direction migrate.MigrationDirection) error {
	suffix := "up.sql"
	if direction == migrate.Down {
		suffix = "down.sql"
	}

	entries, err := clickhouseMigrations.ReadDir("clickhouse_migrations")
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

		content, err := fs.ReadFile(clickhouseMigrations, filepath.Join("clickhouse_migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if _, err := c.db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func (c *ClickHouseStore) Begin() (*sql.Tx, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting ClickHouse: %w", err)
	}

	return tx, nil
}

func (c *ClickHouseStore) Close() error {
	if err := c.db.Close(); err != nil {
		return fmt.Errorf("error closing ClickHouse: %w", err)
	}

	return nil
}
