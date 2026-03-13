package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Kafka       KafkaConfig       `yaml:"kafka"`
	JWT         JWTConfig         `yaml:"jwt"`
	Playwright  PlaywrightConfig  `yaml:"playwright"`
	RateLimiter RateLimiterConfig `yaml:"rate_limiter"`
	HH          HHConfig          `yaml:"hh"`
	Scheduler   SchedulerConfig   `yaml:"scheduler"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type KafkaConfig struct {
	Brokers []string    `yaml:"brokers"`
	Topic   TopicConfig `yaml:"topic"`
}

type TopicConfig struct {
	Vacancies string `yaml:"vacancies"`
}

type JWTConfig struct {
	Secret           string `yaml:"secret"`
	Expiration       int64  `yaml:"expiration"`
	JavaServiceToken string `yaml:"java_service_token"`
}

type PlaywrightConfig struct {
	Headless bool `yaml:"headless"`
	AreaCode int  `yaml:"area_code"`
	SlowMo   int  `yaml:"slow_mo"`
}

type RateLimiterConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	Burst             int  `yaml:"burst"`
}

type HHConfig struct {
	APIURL string `yaml:"api_url"`
}

type SchedulerConfig struct {
	Parser ParserSchedulerConfig `yaml:"parser"`
}

type ParserSchedulerConfig struct {
	Cron string `yaml:"cron"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
