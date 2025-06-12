package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/romanpitatelev/hezzl-goods/internal/configs"
	"github.com/romanpitatelev/hezzl-goods/internal/controller/consumer"
	"github.com/romanpitatelev/hezzl-goods/internal/controller/rest"
	goodshandler "github.com/romanpitatelev/hezzl-goods/internal/controller/rest/goods-handler"
	goodsrepo "github.com/romanpitatelev/hezzl-goods/internal/repository/goods-repo"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/producer"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/store"
	goodsservice "github.com/romanpitatelev/hezzl-goods/internal/usecase/goods-service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
)

func Run(cfg *configs.Config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	log.Level(level)

	db, err := store.New(ctx, store.Config{Dsn: cfg.PostgresDSN})
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to database")
	}

	if err := db.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate")
	}

	clickHouseStore, err := store.NewClickHouse(ctx, cfg.ClickHouseDSN)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to ClickHouse")
	}
	defer clickHouseStore.Close()

	if err = clickHouseStore.Migrate(migrate.Up); err != nil {
		log.Panic().Err(err).Msg("failed to migrate ClickHouse")
	}

	log.Info().Msg("successful migration")

	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Panic().Err(err).Msg("failed to connect to NATS")
	}

	natsConsumer := consumer.New(nc, clickHouseStore)
	if err := natsConsumer.Subscribe(); err != nil {
		log.Panic().Err(err).Msg("failed to subscribe to NATS")
	}

	natsProducer := producer.New(nc, "goods.logs")

	goodsRepo := goodsrepo.New(db)

	goodsService := goodsservice.New(goodsRepo, natsProducer)

	goodsHandler := goodshandler.New(goodsService)

	server := rest.New(
		rest.Config{BindAddress: cfg.BindAddress},
		goodsHandler,
		rest.GetPublicKey(),
	)

	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("failed to run the server: %w", err)
	}

	return nil
}
