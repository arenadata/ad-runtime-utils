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
func TestRun_PrintCACerts_OK(t *testing.T) {
	base := t.TempDir()

	// Fake JAVA_HOME with real cacerts file
	javaHome := filepath.Join(base, "jdk8u")
	cacerts := filepath.Join(javaHome, "lib", "security", "cacerts")
	if err := os.MkdirAll(filepath.Dir(cacerts), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cacerts, []byte("truststore"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(javaHome, "bin", "java"), []byte{}, 0o755); err != nil {
		t.Fatalf("write java: %v", err)
	}
	// <<<

	// Config overriding JAVA_HOME to our temp dir
	cfg := `
default:
  runtimes:
    java:
      version: "8"
      override_path: "` + javaHome + `"
`
	cfgFile := filepath.Join(base, "cfg.yaml")
	if err := os.WriteFile(cfgFile, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	args := []string{"--config", cfgFile, "--runtime", "java", "--print-cacerts"}
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%q", code, errb.String())
	}
	if strings.TrimSpace(out.String()) != cacerts {
		t.Fatalf("stdout=%q want %q", out.String(), cacerts+"\n")
	}
}
func TestRun_PrintCACerts_WrongRuntime(t *testing.T) {
	base := t.TempDir()

	cfg := `
default:
  runtimes:
    python:
      version: "3.11"
      override_path: "` + filepath.Join(base, "py311") + `"
`
	cfgFile := filepath.Join(base, "cfg.yaml")
	_ = os.WriteFile(cfgFile, []byte(cfg), 0o644)

	args := []string{"--config", cfgFile, "--runtime", "python", "--print-cacerts"}
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	if code != exitUserError {
		t.Fatalf("exit=%d want=%d", code, exitUserError)
	}
	if !strings.Contains(errb.String(), "--print-cacerts is only valid with --runtime=java") {
		t.Fatalf("stderr=%q missing guard message", errb.String())
	}
}

func TestRun_PrintCACerts_NotFound(t *testing.T) {
	base := t.TempDir()

	javaHome := filepath.Join(base, "jdk21")
	if err := os.MkdirAll(filepath.Join(javaHome, "lib", "security"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(javaHome, "bin", "java"), []byte{}, 0o755); err != nil {
		t.Fatalf("write java: %v", err)
	}

	cfg := `
default:
  runtimes:
    java:
      version: "21"
      override_path: "` + javaHome + `"
`
	cfgFile := filepath.Join(base, "cfg.yaml")
	if err := os.WriteFile(cfgFile, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	systemCandidates := []string{
		"/etc/ssl/certs/java/cacerts",
		"/etc/pki/ca-trust/extracted/java/cacerts",
	}
	var systemCACerts string
	for _, c := range systemCandidates {
		if st, err := os.Stat(c); err == nil && st.Mode().IsRegular() {
			systemCACerts = c
			break
		}
	}

	args := []string{"--config", cfgFile, "--runtime", "java", "--print-cacerts"}
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)

	if systemCACerts == "" {
		// Expect error branch early-return
		if code != exitUserError {
			t.Fatalf("exit=%d want=%d; stderr=%q", code, exitUserError, errb.String())
		}
		if out.Len() != 0 {
			t.Fatalf("expected empty stdout, got %q", out.String())
		}
		if !strings.Contains(strings.ToLower(errb.String()), "cacerts") {
			t.Fatalf("stderr should mention cacerts error, got %q", errb.String())
		}
		return
	}
	if code != 0 {
		t.Fatalf("exit=%d want=0; stderr=%q", code, errb.String())
	}
	got := strings.TrimSpace(out.String())
	if got != systemCACerts {
		t.Fatalf("stdout=%q want %q", got, systemCACerts)
	}
	if errb.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", errb.String())
	}
}
