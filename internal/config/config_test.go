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
