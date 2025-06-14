package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/rs/zerolog/log"
)

//go:embed clickhouse_migrations
var clickhouseMigrations embed.FS

type ClickHouseStore struct {
	db  *sql.DB
	dsn string
}

func NewClickHouse(ctx context.Context, dsn string) (*ClickHouseStore, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	log.Debug().Msgf("dsn in NewClickHouse() function: %v", dsn)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &ClickHouseStore{
		db:  db,
		dsn: dsn,
	}, nil
}

func (c *ClickHouseStore) Migrate() error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{strings.TrimPrefix(c.dsn, "tcp://")},
		Auth: clickhouse.Auth{
			Database: "hezzl_logs",
			Username: "user",
			Password: "my_pass",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to open native ClickHouse connection: %w", err)
	}
	defer conn.Close()

	entries, err := clickhouseMigrations.ReadDir("clickhouse_migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := fs.ReadFile(clickhouseMigrations, filepath.Join("clickhouse_migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			if err := conn.Exec(context.Background(), stmt); err != nil {
				return fmt.Errorf("failed to execute %s: %w", entry.Name(), err)
			}
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
