package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type App struct {
	Username   string `yaml:"username"`
	SystemName string `yaml:"system_name"`
	Limit      int    `yaml:"limit"`
}

type AppConfig struct {
	Apps []App `yaml:"apps"`
}

func Load(path string) (*AppConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *AppConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil
	}
	return os.WriteFile(path, data, 0644)
}
