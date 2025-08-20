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
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("read config %q: %w", path, readErr)
	}

	var cfg Config
	decodeErr := yaml.UnmarshalWithOptions(data, &cfg, yaml.Strict())
	if decodeErr != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, decodeErr)
	}

	for name, svc := range cfg.Services {
		if svc.Path == "" {
			continue
		}

		extData, readExtErr := os.ReadFile(svc.Path)
		if readExtErr != nil {
			if os.IsNotExist(readExtErr) {
				continue
			}
			return nil, fmt.Errorf("read service config %q: %w", svc.Path, readExtErr)
		}

		var extCfg Config
		fullCfgErr := yaml.UnmarshalWithOptions(extData, &extCfg, yaml.Strict())
		if fullCfgErr == nil {
			if extSvc, found := extCfg.Services[name]; found && len(extSvc.Runtimes) > 0 {
				svc.Runtimes = extSvc.Runtimes
				cfg.Services[name] = svc
				continue
			}
		}

		var ext ServiceConfig
		fallbackErr := yaml.UnmarshalWithOptions(extData, &ext, yaml.Strict())
		if fallbackErr != nil {
			return nil, fmt.Errorf("parse service config %q: %w", svc.Path, fallbackErr)
		}

		svc.Runtimes = ext.Runtimes
		cfg.Services[name] = svc
	}

	return &cfg, nil
}
