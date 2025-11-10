package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arenadata/ad-runtime-utils/internal/config"
	"github.com/arenadata/ad-runtime-utils/internal/detect"
	"github.com/arenadata/ad-runtime-utils/internal/exec"
	"github.com/coreos/go-systemd/v22/daemon"
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
	printCACerts := fs.Bool("print-cacerts", false, "When used with --runtime=java, prints the cacerts path and exits")
	start := fs.Bool("start", false, "Start the service. Use with simple/exec services")
	supervise := fs.Bool("supervise", false, "Supervise the service. Use with notify systemd services")

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

	if *printCACerts {
		if strings.ToLower(*runtime) != "java" {
			fmt.Fprintln(stderr, "--print-cacerts is only valid with --runtime=java")
			return exitUserError
		}
		var javaHome string
		javaHome, err = detect.ResolveRuntime(cfg, *service, "java")
		if err != nil {
			fmt.Fprintf(stderr, "detection failed: %v\n", err)
			return exitUserError
		}
		var cacerts string
		cacerts, err = detect.FindCACerts(javaHome, nil)
		if err != nil {
			fmt.Fprintf(stderr, "cacerts: %v\n", err)
			return exitUserError
		}
		fmt.Fprintln(stdout, cacerts)
		return exitOK
	}

	path, err := detect.ResolveRuntime(cfg, *service, *runtime) // TODO add detectEnvName
	if err != nil {
		fmt.Fprintf(stderr, "detection failed: %v\n", err)
		return exitUserError
	}

	envName := detectEnvName(cfg, *service, *runtime)

	if *start {
		if err = startService(*service, envName, path, *cfg, *supervise); err != nil {
			fmt.Fprintf(stderr, "start service failed: %v\n", err)
			return exitUserError
		}
		return exitOK
	}

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

func startService(service string, envName string, envPath string, cfg config.Config, supervise bool) error {
	srvConfig, ok := cfg.Services[service]
	if !ok {
		return fmt.Errorf("service %s not found in config", service)
	}
	// Append the env for the runtime (eg. JAVA_HOME)
	if srvConfig.EnvVars == nil {
		srvConfig.EnvVars = make(map[string]string)
	}
	srvConfig.EnvVars[envName] = envPath
	if !supervise {
		return exec.RunExecutable(srvConfig.Executable, srvConfig.ExecutableArgs, srvConfig.EnvVars)
	}
	process, err := exec.RunExecutableAsync(srvConfig.Executable, srvConfig.ExecutableArgs, srvConfig.EnvVars)
	if err != nil {
		return err
	}
	// Run the health checks
	for _, checkCfg := range srvConfig.HealthChecks {
		switch checkCfg.Type {
		case exec.PortHealthCheckType:
			portheck := exec.PortHealthCheck{
				PID:    process.Process.Pid,
				Config: checkCfg,
			}
			if err = portheck.Check(); err != nil {
				if err = process.Process.Signal(os.Interrupt); err != nil {
					return fmt.Errorf("failed to send interrupt signal to process: %w", err)
				}
				return fmt.Errorf("health check failed: %w", err)
			}
		default:
			if err = process.Process.Signal(os.Interrupt); err != nil {
				return fmt.Errorf("failed to send interrupt signal to process: %w", err)
			}
			return fmt.Errorf("unknown health check type: %s", checkCfg.Type)
		}
	}
	// Notify systemd daemon that service has started
	if _, err = daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		fmt.Fprintf(os.Stderr, "systemd notification failed: %v\n", err)
	}
	// TODO: Replace this with an actual supervisor loop
	if err = process.Wait(); err != nil {
		return err
	}
	return nil
}
