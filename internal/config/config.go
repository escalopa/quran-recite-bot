package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Redis    RedisConfig    `yaml:"redis"`
	QuranAPI QuranAPIConfig `yaml:"quran_api"`
	App      AppConfig      `yaml:"app"`
}

type TelegramConfig struct {
	Token string `yaml:"token"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type QuranAPIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type AppConfig struct {
	LocalesDir      string `yaml:"locales_dir"`
	DefaultLanguage string `yaml:"default_language"`
}

// Load loads configuration from a YAML file with environment variable overrides
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Override with environment variables if present
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if apiURL := os.Getenv("QURAN_API_URL"); apiURL != "" {
		cfg.QuranAPI.BaseURL = apiURL
	}
	if apiKey := os.Getenv("QURAN_API_KEY"); apiKey != "" {
		cfg.QuranAPI.APIKey = apiKey
	}

	// Validate required fields
	if cfg.Telegram.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}
	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("redis address is required")
	}
	if cfg.QuranAPI.BaseURL == "" {
		return nil, fmt.Errorf("quran API base URL is required")
	}
	if cfg.QuranAPI.APIKey == "" {
		return nil, fmt.Errorf("quran API key is required")
	}

	// Set defaults
	if cfg.App.LocalesDir == "" {
		cfg.App.LocalesDir = "locales"
	}
	if cfg.App.DefaultLanguage == "" {
		cfg.App.DefaultLanguage = "en"
	}

	return &cfg, nil
}
