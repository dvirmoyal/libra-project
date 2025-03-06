package config

import (
	"github.com/Libra-security/devops-server/internal/db"
	"github.com/spf13/viper"
	"strings"
)

// Configuration holds all the configuration settings for the application.
type Configuration struct {
	Server struct {
		Port string
	}
	Statsd struct {
		Host        string
		Port        string
		ServiceName string
	}
	DB *db.DB
}

// LoadConfig loads configuration from a file and environment variables.
func LoadConfig() (*Configuration, error) {
	viper.SetConfigName("conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	// Set environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("LIBRA") // Set the prefix to "LIBRA"
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Configuration
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
