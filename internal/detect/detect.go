package detect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arenadata/ad-runtime-utils/internal/config"
	"github.com/arenadata/ad-runtime-utils/internal/fs"
)

// expandPath expands a leading '~' to the user home directory and
// replaces any environment variables in the path.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	p = os.ExpandEnv(p)
	return p
}

// tryOverridePath checks the cfg.OverridePath (after expansion).
// Returns the expanded path if bin/exe exists there.
func tryOverridePath(cfg config.RuntimeSetting, exe string) (string, bool) {
	if cfg.OverridePath == "" {
		return "", false
	}
	p := expandPath(cfg.OverridePath)
	if _, err := os.Stat(filepath.Join(p, "bin", exe)); err == nil {
		return p, true
	}
	return "", false
}

// tryEnvVar checks the path stored in the environment variable cfg.EnvVar.
// The raw value is expanded before checking.
func tryEnvVar(cfg config.RuntimeSetting, exe string) (string, bool) {
	if cfg.EnvVar == "" {
		return "", false
	}
	raw := os.Getenv(cfg.EnvVar)
	if raw == "" {
		return "", false
	}
	p := expandPath(raw)

	if _, err := os.Stat(filepath.Join(p, "bin", exe)); err == nil {
		return p, true
	}
	return "", false
}

// tryPaths iterates over cfg.Paths, expanding each pattern and performing a reverse-sorted glob.
func tryPaths(cfg config.RuntimeSetting, exe string) (string, bool) {
	for _, pat := range cfg.Paths {
		base := expandPath(pat)

		var globPattern string
		if strings.ContainsAny(base, "*?[") {
			globPattern = base
		} else {
			globPattern = base + "*"
		}
		cands, _ := filepath.Glob(globPattern)
		sort.Sort(sort.Reverse(sort.StringSlice(cands)))
		for _, cand := range cands {
			if fs.ExistsInBin(cand, exe) {
				return cand, true
			}
		}
	}
	return "", false
}

// detectPath applies the three strategies in order: override_path, env_var, and paths.
// Returns the first valid installation directory or false if none found.
func detectPath(cfg config.RuntimeSetting, exe string) (string, bool) {
	if p, ok := tryOverridePath(cfg, exe); ok {
		return p, true
	}
	if p, ok := tryEnvVar(cfg, exe); ok {
		return p, true
	}
	return tryPaths(cfg, exe)
}
