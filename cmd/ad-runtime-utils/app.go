package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/arenadata/ad-runtime-utils/internal/config"
	"github.com/arenadata/ad-runtime-utils/internal/detect"
)

// exit codes.
const (
	exitOK         = 0
	exitUserError  = 1
	exitParseError = 2
)

func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ad-runtime-utils", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfgPath := fs.String("config", "/etc/ad-runtime-utils/adh-runtime-configuration.yaml", "Path to YAML config file")
	service := fs.String("service", "", "Service name (e.g. TRINO)")
	runtime := fs.String("runtime", "", "Runtime to detect (java, python, etc.)")
	listAll := fs.Bool("list", false, "List all detected runtimes (default + services)")
	fs.BoolVar(listAll, "l", false, "shorthand for --list")

	if err := fs.Parse(args); err != nil {
		return exitParseError
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "cannot load config %q: %v\n", *cfgPath, err)
		return exitUserError
	}

	if *listAll {
		return runList(cfg, stdout, stderr)
	}

	if *runtime == "" {
		fmt.Fprintln(stderr, "Error: --runtime is required")
		fs.Usage()
		return exitUserError
	}

	path, err := detect.ResolveRuntime(cfg, *service, *runtime) // TODO add detectEnvName
	if err != nil {
		fmt.Fprintf(stderr, "detection failed: %v\n", err)
		return exitUserError
	}

	envName := detectEnvName(cfg, *service, *runtime)
	fmt.Fprintf(stdout, "export %s=%s\n", envName, path)
	return exitOK
}

func detectEnvName(cfg *config.Config, service, runtime string) string {
	if service != "" {
		if svc, ok := cfg.Services[service]; ok {
			if rtCfg, ok2 := svc.Runtimes[runtime]; ok2 && rtCfg.EnvVar != "" {
				return rtCfg.EnvVar
			}
		}
	}
	if def, ok := cfg.Default.Runtimes[runtime]; ok && def.EnvVar != "" {
		return def.EnvVar
	}
	switch strings.ToLower(runtime) {
	case "java":
		return "JAVA_HOME"
	case "python":
		return "VIRTUAL_ENV"
	default:
		return strings.ToUpper(runtime) + "_HOME"
	}
}

func runList(cfg *config.Config, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "Default runtimes:")
	for rt := range cfg.Default.Runtimes {
		p, err := detect.ResolveRuntime(cfg, "", rt)
		if err != nil {
			fmt.Fprintf(stderr, "  %s: error: %v\n", rt, err)
		} else {
			fmt.Fprintf(stdout, "  %s: %s\n", rt, p)
		}
	}
	for svcName, svcCfg := range cfg.Services {
		fmt.Fprintf(stdout, "\nService %s:\n", svcName)
		for rt := range svcCfg.Runtimes {
			p, err := detect.ResolveRuntime(cfg, svcName, rt)
			if err != nil {
				fmt.Fprintf(stderr, "  %s: error: %v\n", rt, err)
			} else {
				fmt.Fprintf(stdout, "  %s: %s\n", rt, p)
			}
		}
	}
	return exitOK
}
