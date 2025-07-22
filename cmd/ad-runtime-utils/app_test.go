package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_MissingRuntime(t *testing.T) {
	yaml := `
default:
  runtimes:
    java:
      version: "8"
`
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "cfg.yaml")
	if err := os.WriteFile(cfgFile, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out, errb bytes.Buffer
	code := Run([]string{"--config", cfgFile}, &out, &errb)
	if code != exitUserError {
		t.Errorf("Exit code = %d; want %d", code, exitUserError)
	}
	if !bytes.Contains(errb.Bytes(), []byte("--runtime is required")) {
		t.Errorf("stderr %q missing usage", errb.String())
	}
}

func TestRun_ServiceOverride(t *testing.T) {
	base := t.TempDir()
	javaDir := filepath.Join(base, "jdk23")
	os.MkdirAll(filepath.Join(javaDir, "bin"), 0o755)
	os.WriteFile(filepath.Join(javaDir, "bin", "java"), []byte{}, 0o755)

	cfg := `
services:
  svc:
    runtimes:
      java:
        version: "23"
        override_path: "` + javaDir + `"
`
	cfgFile := filepath.Join(base, "cfg.yaml")
	os.WriteFile(cfgFile, []byte(cfg), 0o644)

	args := []string{"--config", cfgFile, "--service", "svc", "--runtime", "java"}
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	if code != 0 {
		t.Fatalf("Exit code = %d; stderr=%q", code, errb.String())
	}
	want := "export JAVA_HOME=" + javaDir + "\n"
	if out.String() != want {
		t.Errorf("stdout = %q; want %q", out.String(), want)
	}
}

func TestRun_ListAll(t *testing.T) {
	base := t.TempDir()
	javaDir := filepath.Join(base, "jdk8")
	os.MkdirAll(filepath.Join(javaDir, "bin"), 0o755)
	os.WriteFile(filepath.Join(javaDir, "bin", "java"), []byte{}, 0o755)

	pyDir := filepath.Join(base, "py39venv")
	os.MkdirAll(filepath.Join(pyDir, "bin"), 0o755)
	os.WriteFile(filepath.Join(pyDir, "bin", "python"), []byte{}, 0o755)

	yaml := `
default:
  runtimes:
    java:
      version: "8"
      override_path: "` + javaDir + `"
services:
  svc:
    runtimes:
      python:
        version: "3.9"
        override_path: "` + pyDir + `"
`
	cfgFile := filepath.Join(base, "cfg.yaml")
	os.WriteFile(cfgFile, []byte(yaml), 0o644)

	args := []string{"--config", cfgFile, "--list"}
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	if code != 0 {
		t.Fatalf("Exit code = %d; stderr=%q", code, errb.String())
	}

	got := out.String()
	if !strings.Contains(got, "Default runtimes:") {
		t.Error("missing 'Default runtimes:' section")
	}
	if !strings.Contains(got, "java: "+javaDir) {
		t.Errorf("missing default java path, got:\n%s", got)
	}
	if !strings.Contains(got, "Service svc:") {
		t.Error("missing 'Service svc:' section")
	}
	if !strings.Contains(got, "python: "+pyDir) {
		t.Errorf("missing service python path, got:\n%s", got)
	}
	if errb.Len() != 0 {
		t.Errorf("expected no stderr, got %q", errb.String())
	}
}

func TestRun_ParseError(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--no-such-flag"}, &out, &errb)
	if code != exitParseError {
		t.Errorf("Exit code = %d; want %d", code, exitParseError)
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"--config", "/no/such/file", "--runtime", "java"}, &out, &errb)
	if code != exitUserError {
		t.Errorf("Exit code = %d; want %d", code, exitUserError)
	}
	if !strings.Contains(errb.String(), "cannot load config") {
		t.Errorf("unexpected stderr: %q", errb.String())
	}
}
