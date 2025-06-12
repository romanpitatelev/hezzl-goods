package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	migrate "github.com/rubenv/sql-migrate"
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

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse")
	}

	return &ClickHouseStore{
		db:  db,
		dsn: dsn,
	}, nil
}

func (c *ClickHouseStore) Migrate(direction migrate.MigrationDirection) error {
	assetSource := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: clickhouseMigrations,
		Root:       "clickhouse_migrations",
	}

	_, err := migrate.Exec(c.db, "clickhouse", assetSource, direction)
	if err != nil {
		return fmt.Errorf("failed to execute ClickHouse migrations: %w", err)
	}

	return nil
}

func (c *ClickHouseStore) Close() error {
	return c.db.Close()
}

func (c *ClickHouseStore) Begin() (*sql.Tx, error) {
	return c.db.Begin()
}
