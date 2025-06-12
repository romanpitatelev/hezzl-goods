package consumer

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/store"
	"github.com/rs/zerolog/log"
)

const (
	defaultBatchSize = 100
	defaultTimout    = 5 * time.Second
)

type NATSConsumer struct {
	clickhouseStore *store.ClickHouseStore
	natsConn        *nats.Conn
	batch           []entity.GoodLog
	batchSize       int
	batchMutex      sync.Mutex
}

func New(natsConn *nats.Conn, clickhouseStore *store.ClickHouseStore) *NATSConsumer {
	nc := &NATSConsumer{
		clickhouseStore: clickhouseStore,
		natsConn:        natsConn,
		batchSize:       defaultBatchSize,
	}

	go func() {
		nc.processBatch(defaultTimout)
	}()

	return nc
}

func (nc *NATSConsumer) Subscribe() error {
	_, err := nc.natsConn.Subscribe("goods.logs", func(msg *nats.Msg) {
		var logMsg entity.GoodLog
		if err := json.Unmarshal(msg.Data, &logMsg); err != nil {
			log.Err(err).Msg("failed to unmarshal message in NATS Subscribe")

			return
		}

		nc.batchMutex.Lock()
		nc.batch = append(nc.batch, logMsg)

		if len(nc.batch) >= nc.batchSize {
			if err := nc.flushBatch(); err != nil {
				log.Warn().Err(err).Msg("failed to flush batch")
			}
		}
		nc.batchMutex.Unlock()
	})

	return fmt.Errorf("error in nats while subscribing: %w", err)
}

func (nc *NATSConsumer) processBatch(d time.Duration) {
	ticker := time.NewTicker(d)
	for range ticker.C {
		nc.batchMutex.Lock()
		if len(nc.batch) > 0 {
			if err := nc.flushBatch(); err != nil {
				continue
			}
		}

		nc.batchMutex.Unlock()
	}
}

//nolint:funlen
func (nc *NATSConsumer) flushBatch() error {
	nc.batchMutex.Lock()
	defer nc.batchMutex.Unlock()

	if len(nc.batch) == 0 {
		return nil
	}

	tx, err := nc.clickhouseStore.Begin()
	if err != nil {
		log.Warn().Err(err).Msg("failed to begin clickhouse")

		return fmt.Errorf("failed to begin clickhouse: %w", err)
	}

	defer func() {
		if err = tx.Rollback(); err != nil {
			log.Warn().Err(err).Msg("failed to rollback transaction in nats consumer")
		}
	}()

	query := `
INSERT INTO goods_logs (id, project_id, name, description, priority, removed, event_time, operation) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)

`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}

	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn().Err(err).Msg("error closing statement")
		}
	}()

	for _, logMsg := range nc.batch {
		_, err = stmt.Exec(
			logMsg.GoodID,
			logMsg.ProjectID,
			logMsg.Name,
			logMsg.Description,
			logMsg.Priority,
			logMsg.Removed,
			logMsg.EventTime,
			logMsg.Operation,
		)
		if err != nil {
			log.Warn().Err(err).Msg("failed to insert log (continuing batch)")

			continue
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	nc.batch = nil

	return nil
}
