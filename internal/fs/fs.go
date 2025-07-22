package fs

import (
	"os"
	"path/filepath"
)

// ExistsInBin checks for path/bin/exe.
func ExistsInBin(path, exe string) bool {
	info, err := os.Stat(filepath.Join(path, "bin", exe))
	return err == nil && !info.IsDir()
}
