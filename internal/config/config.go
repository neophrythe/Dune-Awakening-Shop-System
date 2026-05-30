// Package config loads the Dune Awakening Shop configuration from a YAML file,
// applying environment-variable overrides for secrets.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the full service configuration.
type Config struct {
	ListenAddr string         `yaml:"listen_addr"`
	Database   DatabaseConfig `yaml:"database"`
	Discord    DiscordConfig  `yaml:"discord"`
	Economy    EconomyConfig  `yaml:"economy"`
	Delivery   DeliveryConfig `yaml:"delivery"`
}

// DatabaseConfig points at the Dune game Postgres database.
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	Schema   string `yaml:"schema"`
}

// DiscordConfig configures the bot.
type DiscordConfig struct {
	Token       string `yaml:"token"`
	GuildID     string `yaml:"guild_id"`
	AdminRoleID string `yaml:"admin_role_id"`
}

// EconomyConfig configures playtime rewards.
type EconomyConfig struct {
	CurrencyName    string `yaml:"currency_name"`
	PerMinute       int64  `yaml:"per_minute"`
	AccrualInterval string `yaml:"accrual_interval"` // Go duration string, e.g. "60s"
}

// DeliveryConfig configures in-game item delivery.
type DeliveryConfig struct {
	AMPContainer string `yaml:"amp_container"`
	FLSToken     string `yaml:"fls_token"`
}

// AccrualDuration parses AccrualInterval, falling back to one minute.
func (e EconomyConfig) AccrualDuration() time.Duration {
	if e.AccrualInterval == "" {
		return time.Minute
	}
	d, err := time.ParseDuration(e.AccrualInterval)
	if err != nil || d <= 0 {
		return time.Minute
	}
	return d
}

// Load reads YAML config from path and applies env overrides for secrets.
func Load(path string) (*Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.applyEnv()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		ListenAddr: "0.0.0.0:8090",
		Economy: EconomyConfig{
			CurrencyName:    "Solari",
			PerMinute:       1,
			AccrualInterval: "60s",
		},
	}
}

func (c *Config) applyEnv() {
	if v := os.Getenv("DUNE_SHOP_DISCORD_TOKEN"); v != "" {
		c.Discord.Token = v
	}
	if v := os.Getenv("DUNE_SHOP_DB_PASS"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DUNE_SHOP_FLS_TOKEN"); v != "" {
		c.Delivery.FLSToken = v
	}
}

func (c *Config) validate() error {
	if c.Database.Host == "" || c.Database.Port == 0 {
		return fmt.Errorf("database.host and database.port are required")
	}
	if c.Economy.PerMinute < 0 {
		return fmt.Errorf("economy.per_minute must be >= 0")
	}
	return nil
}
