package configs

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/rs/zerolog/log"
)

const (
	envFileName = "example.env"
)

type Config struct {
	LogLevel string `env:"LOG_Level" env-default:"debug" env-description:"Log level"`

	BindAddress string `env:"BIND_ADDRESS" env-default:":8081" env-description:"Bind address"`
	PostgresDSN string `env:"POSTGRES_DSN" env-default:"postgresql://postgres:my_pass@localhost:5432/hezzl_db" env-description:"PostgreSQL DSN"`

	ClickHouseDSN      string `env:"CLICKHOUSE_DSN" env-default:"tcp://user:my_pass@localhost:9000?database=hezzl_logs" env-description:"ClickHouse DSN"`
	ClickHouseDatabase string `env:"CLICKHOUSE_DATABASE" env-default:"hezzle_logs" env-description:"ClickHouse database name"`

	NATSURL     string `env:"NATS_URL" env-default:"nats://localhost:4222"`
	NATSSubject string `env:"NATS_SUBJECT" env-default:"goods.logs"`

	RedisAddr     string `env:"REDIS_ADDR" env-default:"localhost:6379" env-description:"Redis address"`
	RedisPassword string `env:"REDIS_PASSWORD" env-default:"" env-description:"Redis password"`
	RedisDB       int    `env:"REDIS_DB" env-default:"0" env-description:"Redis database number"`
}

func findConfigFile() bool {
	_, err := os.Stat(envFileName)

	return err == nil
}

func (e *Config) getHelpString() (string, error) {
	baseHeader := "Environment variables that can be set with env: "

	helpString, err := cleanenv.GetDescription(e, &baseHeader)
	if err != nil {
		return "", fmt.Errorf("failed to get help string: %w", err)
	}

	return helpString, nil
}

func New() *Config {
	cfg := &Config{}

	_, err := cfg.getHelpString()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get help string")
	}

	// log.Info().Msg(helpString)

	if findConfigFile() {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			log.Panic().Err(err).Msg("failed to read env config")
		}
	} else if err = cleanenv.ReadConfig(".env", cfg); err != nil {
		log.Panic().Err(err).Msg("failed to read config from .env")
	}

	return cfg
}
