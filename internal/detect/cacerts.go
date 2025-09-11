package detect

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// CACertsOptions allows overriding known system paths (useful for tests).
type CACertsOptions struct {
	KnownSystemPaths []string
}

// DefaultCACertsOptions returns the default options with standard system paths.
func DefaultCACertsOptions() CACertsOptions {
	return CACertsOptions{
		KnownSystemPaths: []string{
			"/etc/pki/ca-trust/extracted/java/cacerts", // RHEL/CentOS
			"/etc/ssl/certs/java/cacerts",              // Debian/Ubuntu
		},
	}
}

// FindCACerts locates the Java truststore ("cacerts") path.
// Search order:
//  1. Under JAVA_HOME (if set):
//     a) symlinks named "cacerts" (resolved to target)
//     b) regular files named "cacerts"
//     c) common locations: $JAVA_HOME/lib/security/cacerts,
//     $JAVA_HOME/jre/lib/security/cacerts
//  2. Known system paths (RHEL/CentOS, Debian/Ubuntu)
func FindCACerts(javaHome string, opts *CACertsOptions) (string, error) {
	o := mergeCACertsOptions(opts)

	// 1) Prefer JAVA_HOME
	if p, ok := findCACertsInJavaHome(javaHome); ok {
		return p, nil
	}

	// 2) Fallback to known system paths
	if p, ok := firstExistingFile(o.KnownSystemPaths); ok {
		return p, nil
	}

	if javaHome == "" {
		return "", errors.New("JAVA_HOME is empty and cacerts not found in system paths")
	}
	return "", fmt.Errorf("unable to find cacerts at JAVA_HOME=%s and in system paths", javaHome)
}

func mergeCACertsOptions(opts *CACertsOptions) CACertsOptions {
	o := DefaultCACertsOptions()
	if opts == nil {
		return o
	}
	if opts.KnownSystemPaths != nil {
		o.KnownSystemPaths = opts.KnownSystemPaths
	}
	return o
}

func findCACertsInJavaHome(javaHome string) (string, bool) {
	if javaHome == "" {
		return "", false
	}
	jh := evalSymlinkOr(javaHome)

	// a) symlinks named "cacerts"
	if p := findSymlinkNamedCACerts(jh); p != "" {
		return p, true
	}
	// b) regular files named "cacerts"
	if p := findFileNamedCACerts(jh); p != "" {
		return p, true
	}
	// c) common direct locations
	if p := firstExistingCommonCACerts(jh); p != "" {
		return p, true
	}
	return "", false
}

func firstExistingCommonCACerts(jh string) string {
	candidates := []string{
		filepath.Join(jh, "lib", "security", "cacerts"),
		filepath.Join(jh, "jre", "lib", "security", "cacerts"),
	}
	for _, p := range candidates {
		if isSymlink(p) {
			if resolved, err := filepath.EvalSymlinks(p); err == nil && isFile(resolved) {
				return resolved
			}
		}
		if isFile(p) {
			return p
		}
	}
	return ""
}

func firstExistingFile(paths []string) (string, bool) {
	for _, p := range paths {
		if isFile(p) {
			return p, true
		}
	}
	return "", false
}

func evalSymlinkOr(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

func isSymlink(p string) bool {
	st, err := os.Lstat(p)
	return err == nil && (st.Mode()&fs.ModeSymlink) != 0
}

// findSymlinkNamedCACerts searches for symlinks named "cacerts" and returns the resolved file target.
func findSymlinkNamedCACerts(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil {
			return nil
		}
		if (d.Type()&fs.ModeSymlink) != 0 && strings.EqualFold(d.Name(), "cacerts") {
			if target, evalErr := filepath.EvalSymlinks(path); evalErr == nil && isFile(target) {
				found = target
				return fs.SkipAll
			}
		}
		return nil
	})
	return found
}

// findFileNamedCACerts searches for regular files named "cacerts".
func findFileNamedCACerts(root string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d == nil {
			return nil
		}
		if d.Type().IsRegular() && strings.EqualFold(d.Name(), "cacerts") {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	return found
}
