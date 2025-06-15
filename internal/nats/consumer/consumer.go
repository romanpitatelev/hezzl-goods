package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/clickhouse"
	"github.com/rs/zerolog/log"
)

const (
	defaultBatchSize = 30
)

type NATSConsumer struct {
	clickhouseStore *clickhouse.Store
	natsConn        *nats.Conn
	batch           []entity.GoodLog
	batchSize       int
	batchMutex      sync.Mutex
}

func New(ctx context.Context, natsConn *nats.Conn, clickhouseStore *clickhouse.Store) *NATSConsumer {
	return &NATSConsumer{
		clickhouseStore: clickhouseStore,
		natsConn:        natsConn,
		batchSize:       defaultBatchSize,
	}
}

func (nc *NATSConsumer) Start() error {
	_, err := nc.natsConn.Subscribe("goods.logs", func(msg *nats.Msg) {
		var logMsg entity.GoodLog
		if err := json.Unmarshal(msg.Data, &logMsg); err != nil {
			log.Err(err).Msg("failed to unmarshal message in NATS Subscribe")

			return
		}

		nc.batchMutex.Lock()
		defer nc.batchMutex.Unlock()

		nc.batch = append(nc.batch, logMsg)

		if len(nc.batch) >= defaultBatchSize {
			batchToFlush := make([]entity.GoodLog, len(nc.batch))
			copy(batchToFlush, nc.batch)
			nc.batch = nc.batch[:0]

			go func() {
				nc.flushBatch(batchToFlush)
			}()
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start nats: %w", err)
	}

	return nil
}

//nolint:funlen
func (nc *NATSConsumer) flushBatch(batch []entity.GoodLog) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := nc.clickhouseStore.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
        INSERT INTO goods_logs (
            id, 
            project_id, 
            name, 
            description, 
            priority, 
            removed, 
            operation, 
            event_time
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, logEntry := range batch {
		_, err = stmt.Exec(
			logEntry.GoodID,
			logEntry.ProjectID,
			logEntry.Name,
			logEntry.Description,
			logEntry.Priority,
			logEntry.Removed,
			logEntry.Operation,
			logEntry.EventTime,
		)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return fmt.Errorf("exec error: %v, rollback error: %w", err, rollbackErr)
			}
			return fmt.Errorf("failed to exec statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Msgf("successfully flushed batch of %d logs to ClickHouse", len(batch))
	return nil
}
