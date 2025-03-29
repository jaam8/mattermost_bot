package tarantool

import (
	"github.com/tarantool/go-tarantool"
	"log"
)

type Config struct {
	Host     string `yaml:"TARANTOOL_HOST" env:"TARANTOOL_HOST" env-default:"localhost"`
	Port     string `yaml:"TARANTOOL_PORT" env:"TARANTOOL_PORT" env-default:"3301"`
	Username string `yaml:"TARANTOOL_USER" env:"TARANTOOL_USER" env-default:"admin"`
	Password string `yaml:"TARANTOOL_PASSWORD" env:"TARANTOOL_PASSWORD" env-default:"secret"`
}

func New(config Config) (*tarantool.Connection, error) {
	conn, err := tarantool.Connect(config.Host+":"+config.Port, tarantool.Opts{
		User: config.Username,
		Pass: config.Password,
	})
	if err != nil {
		log.Fatalf("failed connect to Tarantool: %v", err)
		return nil, err
	}
	return conn, nil
}
