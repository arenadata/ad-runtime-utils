package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	content := []byte(`
default:
  runtimes:
    java:
      version: "8"
      override_path: "/opt/java/default-jdk"
      env_var: JAVA_HOME
autodetect:
  runtimes:
    java:
      "8":
        override_path: "/opt/java/jdk8"
        env_var: JAVA8_HOME
        paths:
          - /usr/lib/jvm/java-1.8*
services:
  svc1:
    runtimes:
      java:
        version: "23"
        override_path: "/opt/java/jdk23"
        env_var: SVC1_JAVA_HOME
`)
	tmp := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	def, ok := cfg.Default.Runtimes["java"]
	if !ok || def.Version != "8" || def.OverridePath != "/opt/java/default-jdk" || def.EnvVar != "JAVA_HOME" {
		t.Errorf("default.java unexpected: %+v", def)
	}

	auto, ok := cfg.Autodetect.Runtimes["java"]["8"]
	if !ok || auto.OverridePath != "/opt/java/jdk8" || auto.EnvVar != "JAVA8_HOME" {
		t.Errorf("autodetect.java.8 unexpected: %+v", auto)
	}

	svc, ok := cfg.Services["svc1"].Runtimes["java"]
	if !ok || svc.Version != "23" || svc.OverridePath != "/opt/java/jdk23" || svc.EnvVar != "SVC1_JAVA_HOME" {
		t.Errorf("services.svc1.java unexpected: %+v", svc)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/no/such/file")
	if err == nil {
		t.Fatal("expected error loading non-existent file")
	}
}

func TestLoad_ServiceExternal(t *testing.T) {
	tmpDir := t.TempDir()
	extContent := []byte(`
services:
  trino:
    runtimes:
      java:
        version: "24"
        env_var: TRINO_JAVA_HOME
      python:
        version: "3.12"
        env_var: TRINO_PY_VENV
`)
	extFile := filepath.Join(tmpDir, "trino.yaml")
	if err := os.WriteFile(extFile, extContent, 0o644); err != nil {
		t.Fatalf("write external service config: %v", err)
	}

	mainContent := []byte(`
default:
  runtimes:
    java:
      version: "8"
autodetect:
  runtimes: {}
services:
  trino:
    path: "` + extFile + `"
`)
	mainFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(mainFile, mainContent, 0o644); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	cfg, err := Load(mainFile)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	svcCfg, ok := cfg.Services["trino"]
	if !ok {
		t.Fatalf("service 'trino' not found")
	}

	java, ok := svcCfg.Runtimes["java"]
	if !ok || java.Version != "24" || java.EnvVar != "TRINO_JAVA_HOME" {
		t.Errorf("external java unexpected: %+v", java)
	}
	py, ok := svcCfg.Runtimes["python"]
	if !ok || py.Version != "3.12" || py.EnvVar != "TRINO_PY_VENV" {
		t.Errorf("external python unexpected: %+v", py)
	}
}

func TestLoad_ServiceExternalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	mainContent := []byte(`
services:
  foo:
    path: "/no/such/file.yaml"
`)
	mainFile := filepath.Join(tmpDir, "cfg.yaml")
	if err := os.WriteFile(mainFile, mainContent, 0o644); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	_, err := Load(mainFile)
	if err == nil {
		t.Fatal("expected error loading missing external service config")
	}
}

func TestLoad_ServiceExternalParseError(t *testing.T) {
	tmpDir := t.TempDir()
	extFile := filepath.Join(tmpDir, "bad.yaml")
	// пишем некорректный YAML
	if err := os.WriteFile(extFile, []byte("::invalid::"), 0o644); err != nil {
		t.Fatalf("write bad external config: %v", err)
	}

	mainContent := []byte(`
services:
  foo:
    path: "` + extFile + `"
`)
	mainFile := filepath.Join(tmpDir, "cfg.yaml")
	if err := os.WriteFile(mainFile, mainContent, 0o644); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	_, err := Load(mainFile)
	if err == nil {
		t.Fatal("expected error parsing external service config")
	}
}
