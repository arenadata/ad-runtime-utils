package detect

import (
	"fmt"

	"github.com/arenadata/ad-runtime-utils/internal/config"
)

func exeName(rt string) string {
	switch rt {
	case "java":
		return "java"
	case "python":
		return "python"
	default:
		return rt
	}
}

func detectServiceLevel(cfg *config.Config, service, runtime, exe string) (string, bool) {
	if service == "" {
		return "", false
	}
	svcCfg, ok := cfg.Services[service]
	if !ok {
		return "", false
	}
	rtCfg, ok := svcCfg.Runtimes[runtime]
	if !ok {
		return "", false
	}
	return detectPath(rtCfg, exe)
}

func detectVersion(cfg *config.Config, service, runtime string) (string, error) {
	if service != "" {
		svcCfg := cfg.Services[service]
		ver := svcCfg.Runtimes[runtime].Version
		if ver == "" {
			return "", fmt.Errorf("version not specified for service '%s' runtime '%s'", service, runtime)
		}
		return ver, nil
	}
	// default
	defCfg := cfg.Default.Runtimes[runtime]
	ver := defCfg.Version
	if ver == "" {
		return "", fmt.Errorf("default version not specified for runtime '%s'", runtime)
	}
	return ver, nil
}

func detectAutodetectVersion(cfg *config.Config, runtime, version, exe string) (string, bool) {
	if versions, ok := cfg.Autodetect.Runtimes[runtime]; ok {
		if verCfg, ok2 := versions[version]; ok2 {
			return detectPath(verCfg, exe)
		}
	}
	return "", false
}

func detectDefault(cfg *config.Config, runtime, exe string) (string, bool) {
	if defCfg, ok := cfg.Default.Runtimes[runtime]; ok {
		return detectPath(defCfg, exe)
	}
	return "", false
}

func ResolveRuntime(cfg *config.Config, service, runtime string) (string, error) {
	exe := exeName(runtime)

	// 1) Service-level detection
	if path, ok := detectServiceLevel(cfg, service, runtime, exe); ok {
		return path, nil
	}

	// 2) Determine version
	version, err := detectVersion(cfg, service, runtime)
	if err != nil {
		return "", err
	}

	// 3) Autodetect per-version
	if path, ok := detectAutodetectVersion(cfg, runtime, version, exe); ok {
		return path, nil
	}

	// 4) Default fallback
	if path, ok := detectDefault(cfg, runtime, exe); ok {
		return path, nil
	}
	return "", fmt.Errorf("could not detect runtime '%s' for service '%s' (version '%s')", runtime, service, version)
}
