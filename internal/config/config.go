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

type ServiceConfig struct {
	Runtimes map[string]RuntimeSetting `yaml:"runtimes,omitempty"`
	Path     string                    `yaml:"path,omitempty"`
}

type Config struct {
	Default struct {
		Runtimes map[string]RuntimeSetting `yaml:"runtimes"`
	} `yaml:"default"`

	Autodetect struct {
		Runtimes map[string]map[string]RuntimeSetting `yaml:"runtimes"`
	} `yaml:"autodetect"`

	Services map[string]ServiceConfig `yaml:"services"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.UnmarshalWithOptions(data, &cfg, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	for name, svc := range cfg.Services {
		if svc.Path != "" {
			extData, err := os.ReadFile(svc.Path)
			if err != nil {
				return nil, fmt.Errorf("read service config %q: %w", svc.Path, err)
			}

			var extFull Config
			if err := yaml.UnmarshalWithOptions(extData, &extFull, yaml.Strict()); err == nil {
				if extSvc, ok := extFull.Services[name]; ok && len(extSvc.Runtimes) > 0 {
					svc.Runtimes = extSvc.Runtimes
					cfg.Services[name] = svc
					continue
				}
			}

			var ext ServiceConfig
			if err := yaml.UnmarshalWithOptions(extData, &ext, yaml.Strict()); err != nil {
				return nil, fmt.Errorf("parse service config %q: %w", svc.Path, err)
			}
			svc.Runtimes = ext.Runtimes
			cfg.Services[name] = svc
		}
	}

	return &cfg, nil
}
