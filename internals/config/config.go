package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Port         int
	MyEmail      string
	GoogleAPIKey string
}

func LoadConfig() (*Config, error) {

	_ = godotenv.Load() // For local development, but env vars take precedence in production

	viper.SetDefault("APP_PORT", "8080")
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.MyEmail == "" {
		cfg.MyEmail = os.Getenv("MY_EMAIL")
		if cfg.MyEmail == "" {
			return nil, &ConfigError{Key: "MY_EMAIL", Value: "", Err: ErrMissingConfig}
		}
	}

	cfg.GoogleAPIKey = os.Getenv("GOOGLE_API_KEY")
	if cfg.GoogleAPIKey == "" {
		return nil, &ConfigError{Key: "GOOGLE_API_KEY", Value: "", Err: ErrMissingConfig}
	}

	return &cfg, nil
}

type ConfigError struct {
	Key   string
	Value string
	Err   error
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Key + " = '" + e.Value + "': " + e.Err.Error()
}

var ErrMissingConfig = os.ErrNotExist
