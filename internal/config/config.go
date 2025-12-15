package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

type RuntimeSetting struct {
	Version      string   `yaml:"version"`
	OverridePath string   `yaml:"override_path,omitempty"`
	EnvVar       string   `yaml:"env_var,omitempty"`
	Paths        []string `yaml:"paths,omitempty"`
}

type HealthCheckConfig struct {
	Type   string         `yaml:"type"`
	Params map[string]any `yaml:"params,omitempty"`
}

type ServiceConfig struct {
	Runtimes       map[string]RuntimeSetting `yaml:"runtimes,omitempty"`
	Path           string                    `yaml:"path,omitempty"`
	Executable     string                    `yaml:"executable,omitempty"`
	ExecutableArgs []string                  `yaml:"executable_args,omitempty"`
	EnvVars        map[string]string         `yaml:"env_vars,omitempty"`
	EnvVarsFile    string                    `yaml:"env_vars_file,omitempty"`
	HealthChecks   []HealthCheckConfig       `yaml:"health_checks,omitempty"`
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

func (h *HealthCheckConfig) ParamToInt(name string) (int, error) {
	val, found := h.Params[name]
	if !found {
		return 0, fmt.Errorf("health check param %q not found", name)
	}
	intVal, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("health check param %q is not an integer", name)
	}
	return intVal, nil
}

func (h *HealthCheckConfig) ParamToString(name string) (string, error) {
	val, found := h.Params[name]
	if !found {
		return "", fmt.Errorf("health check param %q not found", name)
	}
	strVal, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("health check param %q is not a string", name)
	}
	return strVal, nil
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
		// Load Service config from file if specified
		if svc.Path != "" {
			if extCfg, fullCfgErr := parseExternalConfig(svc.Path); fullCfgErr == nil {
				if extSvc, found := extCfg.Services[name]; found {
					cfg.Services[name] = extSvc
					continue
				}
			}

			// Load Service from external file if specified
			ext, err := parseExternalServiceConfig(svc.Path)
			if err != nil {
				return nil, fmt.Errorf("parse external service config %q: %w", svc.Path, err)
			}
			// Replace ServiceConfig with the loaded one
			svc = ext
			// Keep the path to the Service config file for future references
			svc.Path = path
		}

		cfg.Services[name] = svc
	}

	return &cfg, nil
}

func parseExternalConfig(path string) (Config, error) {
	var cfg Config

	extData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read service config %q: %w", path, err)
	}

	if err = yaml.UnmarshalWithOptions(extData, &cfg, yaml.Strict()); err == nil {
		return cfg, nil
	}
	return cfg, nil
}

func parseExternalServiceConfig(path string) (ServiceConfig, error) {
	var cfg ServiceConfig
	extData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read service config %q: %w", path, err)
	}
	if err = yaml.UnmarshalWithOptions(extData, &cfg, yaml.Strict()); err != nil {
		return cfg, fmt.Errorf("parse service config %q: %w", path, err)
	}
	// Abbomination to support env files sourcing
	// It changes exec to bash and adds source command and the original executable as the args to bash
	if cfg.EnvVarsFile != "" {
		argsString := strings.Join(cfg.ExecutableArgs, " ")
		cfg.ExecutableArgs = []string{
			"-c",
			fmt.Sprintf("source %s; %s %s", cfg.EnvVarsFile, cfg.Executable, argsString),
		}
		cfg.Executable = "bash"
	}
	return cfg, nil
}
