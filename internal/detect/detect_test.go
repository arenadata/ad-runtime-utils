package detect

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/arenadata/ad-runtime-utils/internal/config"
	"github.com/arenadata/ad-runtime-utils/internal/fs"
)

func createRuntimeDir(t *testing.T, baseDir, dirName, exe string) string {
	dir := filepath.Join(baseDir, dirName)
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	f, err := os.Create(filepath.Join(binDir, exe))
	if err != nil {
		t.Fatalf("failed to create exe file: %v", err)
	}
	f.Close()
	return dir
}

func TestExistsInBin(t *testing.T) {
	tmp := t.TempDir()
	d := createRuntimeDir(t, tmp, "r1", "java")

	if !fs.ExistsInBin(d, "java") {
		t.Errorf("ExistsInBin returned false for existing java binary")
	}

	if fs.ExistsInBin(d, "javac") {
		t.Errorf("ExistsInBin returned true for non-existent javac")
	}

	if fs.ExistsInBin(filepath.Join(tmp, "no_such"), "java") {
		t.Errorf("ExistsInBin returned true for non-existent path")
	}
}

func TestTryOverridePath(t *testing.T) {
	tmp := t.TempDir()
	valid := createRuntimeDir(t, tmp, "o1", "java")
	rt := config.RuntimeSetting{OverridePath: valid}

	p, ok := tryOverridePath(rt, "java")
	if !ok || p != valid {
		t.Errorf("tryOverridePath = (%q, %v), want (%q, true)", p, ok, valid)
	}

	rt.OverridePath = ""
	if p2, ok2 := tryOverridePath(rt, "java"); ok2 || p2 != "" {
		t.Errorf("tryOverridePath(empty) = (%q, %v), want ('', false)", p2, ok2)
	}
}

func TestTryEnvVar(t *testing.T) {
	tmp := t.TempDir()
	valid := createRuntimeDir(t, tmp, "e1", "py")
	rt := config.RuntimeSetting{EnvVar: "TEST_PY"}

	os.Unsetenv("TEST_PY")
	if p, ok := tryEnvVar(rt, "py"); ok || p != "" {
		t.Errorf("tryEnvVar not set = (%q, %v), want ('', false)", p, ok)
	}

	t.Setenv("TEST_PY", tmp)
	if p, ok := tryEnvVar(rt, "py"); ok || p != "" {
		t.Errorf("tryEnvVar invalid = (%q, %v), want ('', false)", p, ok)
	}

	t.Setenv("TEST_PY", valid)
	if p, ok := tryEnvVar(rt, "py"); !ok || p != valid {
		t.Errorf("tryEnvVar valid = (%q, %v), want (%q, true)", p, ok, valid)
	}
}

func TestTryPaths(t *testing.T) {
	tmp := t.TempDir()
	a := createRuntimeDir(t, tmp, "alpha1", "sh")

	// exact path
	rtExact := config.RuntimeSetting{Paths: []string{a}}
	if p, ok := tryPaths(rtExact, "sh"); !ok || p != a {
		t.Errorf("tryPaths exact = (%q, %v), want (%q, true)", p, ok, a)
	}

	// glob pattern should match a and b, pick b first
	pattern := filepath.Join(tmp, "*")
	rtGlob := config.RuntimeSetting{Paths: []string{pattern}}
	p, ok := tryPaths(rtGlob, "sh")
	if !ok {
		t.Fatalf("tryPaths glob failed")
	}

	cands, _ := filepath.Glob(pattern)
	sort.Sort(sort.Reverse(sort.StringSlice(cands)))
	expected := cands[0]

	if p != expected {
		t.Errorf("tryPaths glob = %q, want %q", p, expected)
	}

	// no paths
	rtEmpty := config.RuntimeSetting{Paths: nil}
	if p2, ok2 := tryPaths(rtEmpty, "sh"); ok2 || p2 != "" {
		t.Errorf("tryPaths empty = (%q, %v), want ('', false)", p2, ok2)
	}
}

func TestDetectPath_Order(t *testing.T) {
	tmp := t.TempDir()
	o := createRuntimeDir(t, tmp, "o", "exe")
	e := createRuntimeDir(t, tmp, "e", "exe")
	p := createRuntimeDir(t, tmp, "p", "exe")

	rt := config.RuntimeSetting{
		OverridePath: o,
		EnvVar:       "TEST_EXE",
		Paths:        []string{p},
	}
	t.Setenv("TEST_EXE", e)

	// override wins
	if got, _ := detectPath(rt, "exe"); got != o {
		t.Errorf("detectPath override = %q, want %q", got, o)
	}

	rt.OverridePath = ""
	// env next
	if got, _ := detectPath(rt, "exe"); got != e {
		t.Errorf("detectPath env = %q, want %q", got, e)
	}

	// path next
	os.Unsetenv("TEST_EXE")
	rt.EnvVar = ""
	if got, _ := detectPath(rt, "exe"); got != p {
		t.Errorf("detectPath path = %q, want %q", got, p)
	}

	// empty
	rt.Paths = nil
	if got, ok := detectPath(rt, "exe"); ok {
		t.Errorf("detectPath empty = (%q, %v), want ('', false)", got, ok)
	}
}
