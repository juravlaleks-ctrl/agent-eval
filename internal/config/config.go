package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed default_config.yaml
var defaultConfigYAML []byte

// Config is the top-level configuration structure.
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Session     SessionConfig     `yaml:"session"`
	Tasks       TasksConfig       `yaml:"tasks"`
	Replication ReplicationConfig `yaml:"replication"`
	Profiles    []Profile         `yaml:"profiles"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig holds SQLite database settings.
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// SessionConfig holds session lifecycle settings.
type SessionConfig struct {
	IdleTimeoutMinutes int `yaml:"idle_timeout_minutes"`
}

// TasksConfig holds task-selection settings.
type TasksConfig struct {
	Randomize bool `yaml:"randomize"`
}

// ReplicationConfig holds subagent replication settings.
type ReplicationConfig struct {
	Enabled bool `yaml:"enabled"`
	Count   int  `yaml:"count"`
}

// Profile maps a (model, effort) pair to a difficulty range.
type Profile struct {
	Match      ProfileMatch `yaml:"match"`
	Difficulty [2]int       `yaml:"difficulty"` // [min, max] inclusive
}

// ProfileMatch is the key used to look up a profile.
type ProfileMatch struct {
	Model  string `yaml:"model"`
	Effort string `yaml:"effort"`
}

// Load reads YAML configuration from path. If path is empty or the file does
// not exist, the embedded default_config.yaml is used instead.
func Load(path string) (*Config, error) {
	var data []byte

	if path == "" {
		data = defaultConfigYAML
	} else {
		raw, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				data = defaultConfigYAML
			} else {
				return nil, fmt.Errorf("config: read %q: %w", path, err)
			}
		} else {
			data = raw
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse YAML: %w", err)
	}
	return &cfg, nil
}

// Validate checks that all configuration values are in range and consistent.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("config: server.port %d out of range [1, 65535]", c.Server.Port)
	}
	if c.Session.IdleTimeoutMinutes < 1 {
		return fmt.Errorf("config: session.idle_timeout_minutes must be >= 1, got %d", c.Session.IdleTimeoutMinutes)
	}
	if c.Replication.Enabled && c.Replication.Count < 1 {
		return fmt.Errorf("config: replication.count must be >= 1 when replication is enabled, got %d", c.Replication.Count)
	}

	validEfforts := map[string]bool{"low": true, "medium": true, "high": true}
	for i, p := range c.Profiles {
		if p.Match.Model == "" {
			return fmt.Errorf("config: profile[%d]: model must not be empty", i)
		}
		if p.Match.Effort == "" {
			return fmt.Errorf("config: profile[%d]: effort must not be empty", i)
		}
		if !validEfforts[p.Match.Effort] {
			return fmt.Errorf("config: profile[%d]: effort %q must be one of low, medium, high", i, p.Match.Effort)
		}
		if p.Difficulty[0] < 1 || p.Difficulty[0] > 3 {
			return fmt.Errorf("config: profile[%d]: difficulty[0] %d out of range [1, 3]", i, p.Difficulty[0])
		}
		if p.Difficulty[1] < 1 || p.Difficulty[1] > 3 {
			return fmt.Errorf("config: profile[%d]: difficulty[1] %d out of range [1, 3]", i, p.Difficulty[1])
		}
		if p.Difficulty[0] > p.Difficulty[1] {
			return fmt.Errorf("config: profile[%d]: difficulty[0] %d > difficulty[1] %d", i, p.Difficulty[0], p.Difficulty[1])
		}
	}
	return nil
}
