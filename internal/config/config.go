package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Redis    RedisConfig    `mapstructure:"redis"`
	QuranAPI QuranAPIConfig `mapstructure:"quran_api"`
	App      AppConfig      `mapstructure:"app"`
}

type TelegramConfig struct {
	Token string `mapstructure:"token"`
}

type RedisConfig struct {
	URI string `mapstructure:"uri"`
}

type QuranAPIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

type AppConfig struct {
	LocalesDir      string `mapstructure:"locales_dir"`
	DefaultLanguage string `mapstructure:"default_language"`
}

// Load loads configuration from a YAML file with environment variable overrides
func Load(filename string) (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(filename)

	// Set defaults
	v.SetDefault("app.locales_dir", "locales")
	v.SetDefault("app.default_language", "en")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Environment variable configuration
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.Telegram.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}
	if cfg.Redis.URI == "" {
		return nil, fmt.Errorf("redis URI is required")
	}
	if cfg.QuranAPI.BaseURL == "" {
		return nil, fmt.Errorf("quran API base URL is required")
	}
	if cfg.QuranAPI.APIKey == "" {
		return nil, fmt.Errorf("quran API key is required")
	}

	return &cfg, nil
}
