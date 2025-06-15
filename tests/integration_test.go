package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/romanpitatelev/hezzl-goods/internal/controller/rest"
	goodshandler "github.com/romanpitatelev/hezzl-goods/internal/controller/rest/goods-handler"
	"github.com/romanpitatelev/hezzl-goods/internal/nats/consumer"
	"github.com/romanpitatelev/hezzl-goods/internal/nats/producer"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/clickhouse"
	goodsrepo "github.com/romanpitatelev/hezzl-goods/internal/repository/goods-repo"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/postgres"
	"github.com/romanpitatelev/hezzl-goods/internal/repository/redis"
	goodsservice "github.com/romanpitatelev/hezzl-goods/internal/usecase/goods-service"
	"github.com/rs/zerolog/log"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
)

const (
	pgDSN         = "postgresql://postgres:my_pass@localhost:5432/hezzl_db"
	clickhouseDSN = "clickhouse://user:my_pass@localhost:9000/hezzl_logs"
	natsURL       = "nats://localhost:4222"
	redisAddr     = "localhost:6379"
	redisPassword = ""
	redisDB       = 0
	port          = 5003
	goodsPath     = "/api/v1/good"
)

type IntegrationTestSuite struct {
	suite.Suite
	cancelFunc      context.CancelFunc
	db              *postgres.DataStore
	clickhouseStore *clickhouse.Store
	natsClient      *consumer.NATSConsumer
	natsProducer    *producer.NatsWrapper
	redisClient     *redis.Client
	goodsrepo       *goodsrepo.Repo
	goodsservice    *goodsservice.Service
	goodshandler    *goodshandler.Handler
	server          *rest.Server
}

func (s *IntegrationTestSuite) SetupSuite() {
	log.Info().Msg("starting SetupSuite ...")

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFunc = cancel

	var err error

	s.db, err = postgres.New(ctx, postgres.Config{Dsn: pgDSN})
	s.Require().NoError(err)

	log.Info().Msg("starting postgres ...")

	err = s.db.Migrate(migrate.Up)
	s.Require().NoError(err)

	log.Info().Msg("postgres migrations ready")

	s.clickhouseStore, err = clickhouse.New(ctx, clickhouse.Config{Dsn: clickhouseDSN})
	s.Require().NoError(err)

	defer func() {
		err := s.clickhouseStore.Close()
		s.Require().NoError(err)
	}()

	err = s.clickhouseStore.Migrate(migrate.Up)
	s.Require().NoError(err)

	log.Info().Msg("successful ClickHouse migration")

	nc, err := nats.Connect(natsURL)
	s.Require().NoError(err)

	status := nc.Status()
	s.Require().Equal(nats.CONNECTED, status)

	log.Info().Msg("successful connection to NATS")

	s.natsClient = consumer.New(nc, s.clickhouseStore)
	err = s.natsClient.Subscribe()
	s.Require().NoError(err)

	s.natsProducer = producer.New(nc, "goods.logs")

	s.goodsrepo = goodsrepo.New(s.db)

	s.redisClient, err = redis.New(ctx, redisAddr, redisPassword, redisDB)
	s.Require().NoError(err)

	defer func() {
		err := s.redisClient.Close()
		s.Require().NoError(err)
	}()

	s.goodsservice = goodsservice.New(s.goodsrepo, s.natsProducer, s.redisClient)

	s.goodshandler = goodshandler.New(s.goodsservice)

	s.server = rest.New(
		rest.Config{BindAddress: fmt.Sprintf(":%d", port)},
		s.goodshandler,
	)

	//nolint:testifylint
	go func() {
		err = s.server.Run(ctx)
		s.Require().NoError(err)
	}()

	time.Sleep(50 * time.Millisecond)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.cancelFunc()
}

func (s *IntegrationTestSuite) TearDownTest() {
	err := s.db.Truncate(context.Background(),
		"goods",
		"projects",
	)
	s.Require().NoError(err)

	err = s.clickhouseStore.Truncate(context.Background(),
		"goods_logs",
	)
	s.Require().NoError(err)
}

func TestIntegrationSetupSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) sendRequest(method, path string, status int, entity, result any) {
	body, err := json.Marshal(entity)
	s.Require().NoError(err)

	requestURL := fmt.Sprintf("http://localhost:%d%s", port, path)
	s.T().Logf("Sending request to %s", requestURL)

	request, err := http.NewRequestWithContext(context.Background(), method,
		fmt.Sprintf("http://localhost:%d%s", port, path), bytes.NewReader(body))
	s.Require().NoError(err, "fail to create request")

	client := http.Client{}

	response, err := client.Do(request)

	s.Require().NoError(err, "fail to execute request")

	s.Require().NotNil(response, "response object is nil")

	defer func() {
		err = response.Body.Close()
		s.Require().NoError(err)
	}()

	s.T().Logf("Response Status Code: %d", response.StatusCode)

	if status != response.StatusCode {
		responseBody, err := io.ReadAll(response.Body)
		s.Require().NoError(err)

		s.T().Logf("Response Body: %s", string(responseBody))

		s.Require().Equal(status, response.StatusCode, "unexpected status code")

		return
	}

	if result == nil {
		return
	}

	err = json.NewDecoder(response.Body).Decode(result)
	s.Require().NoError(err)
}
