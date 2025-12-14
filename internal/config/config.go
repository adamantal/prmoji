package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	SlackToken        string
	Port              int
	LogLevel          string
	IgnoredCommenters []string
	RetentionDays     int
	DBPath            string
}

func Load() (Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("PORT", 5000)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("RETENTION_DAYS", 90)
	v.SetDefault("DB_PATH", "./prmoji.db")
	v.SetDefault("IGNORED_COMMENTERS", "")

	cfg := Config{
		SlackToken:    v.GetString("SLACK_TOKEN"),
		Port:          v.GetInt("PORT"),
		LogLevel:      v.GetString("LOG_LEVEL"),
		RetentionDays: v.GetInt("RETENTION_DAYS"),
		DBPath:        v.GetString("DB_PATH"),
	}

	cfg.IgnoredCommenters = strings.Split(v.GetString("IGNORED_COMMENTERS"), ",")

	if strings.TrimSpace(cfg.SlackToken) == "" {
		return Config{}, errors.New("SLACK_TOKEN is required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT: %d", cfg.Port)
	}
	if cfg.RetentionDays <= 0 {
		return Config{}, fmt.Errorf("invalid RETENTION_DAYS: %d", cfg.RetentionDays)
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		return Config{}, errors.New("DB_PATH cannot be empty")
	}

	return cfg, nil
}
