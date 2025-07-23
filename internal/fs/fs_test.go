package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arenadata/ad-runtime-utils/internal/fs"
)

// createExeDir creates base/relPath/bin/exe and returns the directory containing bin.
func createExeDir(t *testing.T, base, relPath, exe string) string {
	dir := filepath.Join(base, relPath)
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	f, err := os.Create(filepath.Join(bin, exe))
	if err != nil {
		t.Fatalf("failed to create exe file: %v", err)
	}
	f.Close()
	return dir
}

func TestExistsInBin_DirectBin(t *testing.T) {
	tmp := t.TempDir()
	d := createExeDir(t, tmp, "r1", "java")
	// Should find bin/java under r1
	if !fs.ExistsInBin(tmp, "java") {
		t.Errorf("ExistsInBin failed to find direct bin for %q", d)
	}
}

func TestExistsInBin_NestedJre(t *testing.T) {
	tmp := t.TempDir()
	d := createExeDir(t, tmp, filepath.Join("r2", "jre"), "java")
	// Should find bin/java under r2/jre
	if !fs.ExistsInBin(tmp, "java") {
		t.Errorf("ExistsInBin failed to find nested jre bin for %q", d)
	}
}

func TestExistsInBin_DeeperSubdir(t *testing.T) {
	tmp := t.TempDir()
	d := createExeDir(t, tmp, filepath.Join("r3", "sdk"), "java")
	// Should find bin/java under r3/sdk
	if !fs.ExistsInBin(tmp, "java") {
		t.Errorf("ExistsInBin failed to find deeper bin for %q", d)
	}
}

func TestExistsInBin_NotExists(t *testing.T) {
	tmp := t.TempDir()
	// No exe anywhere
	if fs.ExistsInBin(tmp, "nonexistent") {
		t.Error("ExistsInBin returned true for missing exe")
	}
}
