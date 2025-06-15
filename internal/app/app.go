package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/romanpitatelev/hezzl-goods/internal/configs"
	"github.com/romanpitatelev/hezzl-goods/internal/controller/rest"
	goodshandler "github.com/romanpitatelev/hezzl-goods/internal/controller/rest/goods-handler"
	"github.com/romanpitatelev/hezzl-goods/internal/nats/consumer"
	"github.com/romanpitatelev/hezzl-goods/internal/nats/producer"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/clickhouse"
	goodsrepo "github.com/romanpitatelev/hezzl-goods/internal/repository/goods-repo"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/postgres"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/redis"
	goodsservice "github.com/romanpitatelev/hezzl-goods/internal/usecase/goods-service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

//nolint:funlen
func Run(cfg *configs.Config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	log.Level(level)

	db, err := postgres.New(ctx, postgres.Config{Dsn: cfg.PostgresDSN})
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to database")
	}

	if err := db.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate")
	}

	log.Info().Msg("successful Postgres migration")

	clickHouseStore, err := clickhouse.New(ctx, clickhouse.Config{Dsn: cfg.ClickHouseDSN})
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to ClickHouse")
	}

	defer func() {
		if err := clickHouseStore.Close(); err != nil {
			log.Warn().Err(err).Msg("failed to close ClickHouse connection")
		}
	}()

	if err = clickHouseStore.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate ClickHouse")
	}

	log.Info().Msg("successful ClickHouse migration")

	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to NATS")
	}

	if status := nc.Status(); status != nats.CONNECTED {
		log.Panic().Msgf("NATS connection status: %v", status)
	}

	log.Info().Msg("successful connection to NATS")

	natsConsumer := consumer.New(nc, clickHouseStore)
	if err := natsConsumer.Subscribe(); err != nil {
		log.Panic().Err(err).Msg("failed to subscribe to NATS")
	}

	natsProducer := producer.New(nc, "goods.logs")

	goodsRepo := goodsrepo.New(db)

	redisClient, err := redis.New(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to Redis")
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Warn().Err(err).Msg("error closing Redis connection")
		}
	}()

	goodsService := goodsservice.New(goodsRepo, natsProducer, redisClient)

	goodsHandler := goodshandler.New(goodsService)

	server := rest.New(
		rest.Config{BindAddress: cfg.BindAddress},
		goodsHandler,
	)

	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("failed to run the server: %w", err)
	}

	return nil
}
