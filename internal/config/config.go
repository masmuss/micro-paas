// Package config provides configuration loading for the application.
package config

import (
	"errors"

	"github.com/spf13/viper"
)

// Config holds the application configuration loaded from file or environment variables.
// All fields are required for the application to run.
type Config struct {
	// ServerPort is the port on which the HTTP server will listen (e.g. "8080").
	ServerPort string `mapstructure:"SERVER_PORT" validate:"required"`
	// DBPath is the file path to the SQLite database (e.g. "./micro-paas.db").
	DBPath string `mapstructure:"DB_PATH" validate:"required"`
	// DockerSocket is the path to the Docker socket for container management (e.g. "/var/run/docker.sock").
	DockerSocket string `mapstructure:"DOCKER_SOCKET" validate:"required"`
}

// LoadConfig loads configuration from config.yaml (if present) and environment variables.
// It returns a Config struct and an error if loading or unmarshalling fails.
// Default values are set for all fields if not provided.
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
