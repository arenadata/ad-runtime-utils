package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type RuntimeSetting struct {
	Version      string   `yaml:"version"`
	OverridePath string   `yaml:"override_path,omitempty"`
	EnvVar       string   `yaml:"env_var,omitempty"`
	Paths        []string `yaml:"paths,omitempty"`
}

type Config struct {
	Default struct {
		Runtimes map[string]RuntimeSetting `yaml:"runtimes"`
	} `yaml:"default"`

	Autodetect struct {
		Runtimes map[string]map[string]RuntimeSetting `yaml:"runtimes"`
	} `yaml:"autodetect"`

	Services map[string]struct {
		Runtimes map[string]RuntimeSetting `yaml:"runtimes"`
	} `yaml:"services"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
