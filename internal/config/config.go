package config

import (
	"errors"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort   string `mapstructure:"SERVER_PORT"   validate:"required"`
	DBPath       string `mapstructure:"DB_PATH"       validate:"required"`
	DockerSocket string `mapstructure:"DOCKER_SOCKET" validate:"required"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("DB_PATH", "./micro-paas.db")
	viper.SetDefault("DOCKER_SOCKET", "/var/run/docker.sock")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, err
		}
	}

	var config Config
	err := viper.Unmarshal(&config)

	return &config, err
}
