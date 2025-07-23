package fs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

var errFound = errors.New("found")

// ExistsInBin recursively searches under root (including root itself)
// for any directory containing bin/exe. It returns true on first match.
func ExistsInBin(root, exe string) bool {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// propagate FS error to abort traversal
			return err
		}
		if !d.IsDir() {
			return nil
		}
		binExe := filepath.Join(path, "bin", exe)
		if info, err2 := os.Stat(binExe); err2 == nil && !info.IsDir() {
			return errFound // found, stop walk
		}
		return nil
	})
	return errors.Is(err, errFound)
}
