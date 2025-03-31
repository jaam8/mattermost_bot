package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jaam8/mattermost_bot/pkg/tarantool"
	"github.com/joho/godotenv"
)

type Config struct {
	RestPort  string           `yaml:"REST_PORT"  env:"REST_PORT" env-default:"8080"`
	BotToken  string           `yaml:"BOT_TOKEN"  env:"BOT_TOKEN"`
	MmURL     string           `yaml:"MM_URL"     env:"MM_URL"`
	MmWsURL   string           `yaml:"MM_WS_URL"  env:"MM_WS_URL"`
	LogLevel  string           `yaml:"LOG_LEVEL"  env:"LOG_LEVEL" env-default:"debug"`
	Tarantool tarantool.Config `yaml:"TARANTOOL"  env:"TARANTOOL"`
}

func New() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	var config Config
	if err := cleanenv.ReadEnv(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
