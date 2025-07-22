package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arenadata/ad-runtime-utils/internal/config"
)

// writeYAML writes s to a temp file and returns its path.
func writeYAML(t *testing.T, s string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(f, []byte(s), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return f
}

// makeDir creates base/name/bin/exe and returns base/name.
func makeDir(t *testing.T, base, name, exe string) string {
	t.Helper()
	dir := filepath.Join(base, name)
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	f, err := os.Create(filepath.Join(bin, exe))
	if err != nil {
		t.Fatalf("create exe failed: %v", err)
	}
	f.Close()
	return dir
}

func TestResolveRuntime(t *testing.T) {
	base := t.TempDir()
	java23 := makeDir(t, base, "jdk23", "java")
	py39 := makeDir(t, base, "py39venv", "python")
	// write a YAML combining all cases
	yaml := `
default:
  runtimes:
    java:
      version: "8"
    python:
      version: "3.9"
autodetect:
  runtimes:
    java:
      "8":
        paths:
          - "` + base + `/jdk8*"
    python:
      "3.9":
        paths:
          - "` + base + `/py39*"
services:
  svc1:
    runtimes:
      java:
        version: "23"
        override_path: "` + java23 + `"
      python:
        version: "3.9"
        env_var: SVC1_PY
`
	cfgFile := writeYAML(t, yaml)
	cfg, err := config.Load(cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// ensure environment var case
	t.Setenv("SVC1_PY", py39)
	defer os.Unsetenv("SVC1_PY")

	tests := []struct {
		name    string
		service string
		runtime string
		wantDir string
		wantErr bool
	}{
		{
			name:    "service java override_path",
			service: "svc1", runtime: "java",
			wantDir: java23,
		},
		{
			name:    "service python env_var",
			service: "svc1", runtime: "python",
			wantDir: py39,
		},
		{
			name:    "default java autodetect",
			service: "", runtime: "java",
			// only autodetect for 8, jdk23 not used in default flow
			wantDir: "", wantErr: true,
		},
		{
			name:    "default python autodetect",
			service: "", runtime: "python",
			// picks py39 via glob
			wantDir: py39,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, runErr := ResolveRuntime(cfg, tc.service, tc.runtime)
			if tc.wantErr {
				if runErr == nil {
					t.Fatalf("expected error, got none, dir=%q", got)
				}
				return
			}
			if runErr != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantDir {
				t.Errorf("got %q, want %q", got, tc.wantDir)
			}
		})
	}
}
