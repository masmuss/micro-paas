// Package config provides configuration loading for the application.
package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the application configuration loaded from file or environment variables.
// All fields are required for the application to run.
type Config struct {
	// ServerPort is the port on which the HTTP server will listen (e.g. "8080").
	ServerPort string `mapstructure:"server_port"`
	// DBDriver is the database driver to use (sqlite, mysql, postgres).
	DBDriver string `mapstructure:"db_driver"`
	// DBDsn is the Data Source Name for the database connection.
	DBDsn string `mapstructure:"db_dsn"`
	// DockerSocket is the path to the Docker socket for container management (e.g. "/var/run/docker.sock").
	DockerSocket string `mapstructure:"docker_socket"`
	// MainDomain is the primary domain for the dashboard (e.g. "micro-paas.local").
	MainDomain string `mapstructure:"main_domain"`
}

// LoadConfig loads configuration from config.yml (if present) and environment variables.
// It returns a Config struct and an error if loading or unmarshalling fails.
func LoadConfig() (*Config, error) {
	viper.SetDefault("server_port", "8080")
	viper.SetDefault("db_driver", "sqlite")
	viper.SetDefault("db_dsn", "micro-paas.db")
	viper.SetDefault("docker_socket", "/var/run/docker.sock")
	viper.SetDefault("main_domain", "localhost")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Manual validation for required fields
	if config.ServerPort == "" {
		return nil, errors.New("server_port is required")
	}
	if config.DBDriver == "" {
		return nil, errors.New("db_driver is required")
	}
	if config.DBDsn == "" {
		return nil, errors.New("db_dsn is required")
	}
	if config.DockerSocket == "" {
		return nil, errors.New("docker_socket is required")
	}

	return &config, nil
}
