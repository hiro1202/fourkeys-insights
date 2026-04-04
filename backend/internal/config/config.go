package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	GitHub GitHubConfig `mapstructure:"github"`
	Log    LogConfig    `mapstructure:"log"`
	Fetch  FetchConfig  `mapstructure:"fetch"`
}

type AppConfig struct {
	Port int    `mapstructure:"port"`
	Bind string `mapstructure:"bind"`
}

type GitHubConfig struct {
	Token      string `mapstructure:"token"`
	APIBaseURL string `mapstructure:"api_base_url"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type FetchConfig struct {
	Concurrency int `mapstructure:"concurrency"`
}

func Load() (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.bind", "localhost")
	v.SetDefault("github.token", "")
	v.SetDefault("github.api_base_url", "https://api.github.com")
	v.SetDefault("log.level", "info")
	v.SetDefault("fetch.concurrency", 1)

	// Config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	// Environment variables: GITHUB_TOKEN, APP_PORT, etc.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// GITHUB_TOKEN env var takes priority
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		v.Set("github.token", token)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
