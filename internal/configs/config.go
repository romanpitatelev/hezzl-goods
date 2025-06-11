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
	PostgresDSN string `env:"POSTGRES_DSN" env-default:"postgresql://postgres:my_pass@localhost:5432/denet_db" env-description:"PostgreSQL DSN"`
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

	helpString, err := cfg.getHelpString()
	if err != nil {
		log.Panic().Err(err).Msg("failed to get help string")
	}

	log.Info().Msg(helpString)

	if findConfigFile() {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			log.Panic().Err(err).Msg("failed to read env config")
		}
	} else if err = cleanenv.ReadConfig(".env", cfg); err != nil {
		log.Panic().Err(err).Msg("failed to read config from .env")
	}

	return cfg
}
