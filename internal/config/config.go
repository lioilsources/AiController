package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ModelConfig struct {
	ComposeFile string `yaml:"compose_file"`
	HealthURL   string `yaml:"health_url"`
	Alias       string `yaml:"alias"`
}

type Config struct {
	Models map[string]ModelConfig `yaml:"models"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = "config/models.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if len(cfg.Models) == 0 {
		return nil, fmt.Errorf("config %s: no models defined", path)
	}
	return &cfg, nil
}
