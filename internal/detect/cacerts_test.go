package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: must create a regular file with contents
func mustWriteFile(t *testing.T, p string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

// helper: create a symlink
func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(link), err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
}

func optsNoSystem() *CACertsOptions {
	o := DefaultCACertsOptions()
	o.KnownSystemPaths = []string{}
	return &o
}

func optsWithSystem(paths ...string) *CACertsOptions {
	return &CACertsOptions{KnownSystemPaths: paths}
}

func TestFindCACerts_CommonLocation_File(t *testing.T) {
	tmp := t.TempDir()
	javaHome := tmp

	// $JAVA_HOME/lib/security/cacerts
	cacerts := filepath.Join(javaHome, "lib", "security", "cacerts")
	mustWriteFile(t, cacerts, []byte("truststore"))

	got, err := FindCACerts(javaHome, optsNoSystem())
	if err != nil {
		t.Fatalf("FindCACerts error: %v", err)
	}
	if got != cacerts {
		t.Fatalf("got %q, want %q", got, cacerts)
	}
}

func TestFindCACerts_CommonLocation_Symlink(t *testing.T) {
	tmp := t.TempDir()
	javaHome := tmp

	realPath := filepath.Join(tmp, "real_cacerts")
	mustWriteFile(t, realPath, []byte("truststore"))
	link := filepath.Join(javaHome, "lib", "security", "cacerts")
	mustSymlink(t, realPath, link)

	got, err := FindCACerts(javaHome, optsNoSystem())
	if err != nil {
		t.Fatalf("FindCACerts error: %v", err)
	}
	if got != realPath {
		t.Fatalf("got %q, want resolved %q", got, realPath)
	}
}

func TestFindCACerts_SymlinkNamedCACerts_Deep(t *testing.T) {
	tmp := t.TempDir()
	javaHome := tmp

	realPath := filepath.Join(tmp, "ts", "cacerts.real")
	mustWriteFile(t, realPath, []byte("truststore"))

	// create a deep symlink named exactly "cacerts" somewhere under JAVA_HOME
	symlinkPath := filepath.Join(javaHome, "some", "deep", "path", "cacerts")
	mustSymlink(t, realPath, symlinkPath)

	got, err := FindCACerts(javaHome, optsNoSystem())
	if err != nil {
		t.Fatalf("FindCACerts error: %v", err)
	}
	if got != realPath {
		t.Fatalf("got %q, want resolved %q", got, realPath)
	}
}

func TestFindCACerts_FileNamedCACerts_Deep(t *testing.T) {
	tmp := t.TempDir()
	javaHome := tmp

	deepFile := filepath.Join(javaHome, "another", "deep", "dir", "cacerts")
	mustWriteFile(t, deepFile, []byte("truststore"))

	got, err := FindCACerts(javaHome, optsNoSystem())
	if err != nil {
		t.Fatalf("FindCACerts error: %v", err)
	}
	if got != deepFile {
		t.Fatalf("got %q, want %q", got, deepFile)
	}
}

func TestFindCACerts_KnownSystemPath(t *testing.T) {
	tmp := t.TempDir()
	sys := filepath.Join(tmp, "etc", "ssl", "certs", "java", "cacerts")
	mustWriteFile(t, sys, []byte("truststore"))

	// Even if JAVA_HOME is empty, system path should be discovered
	got, err := FindCACerts("", optsWithSystem(sys))
	if err != nil {
		t.Fatalf("FindCACerts error: %v", err)
	}
	if got != sys {
		t.Fatalf("got %q, want %q", got, sys)
	}
}

func TestFindCACerts_EmptyJavaHome_Error(t *testing.T) {
	if _, err := FindCACerts("", optsNoSystem()); err == nil {
		t.Fatalf("expected error for empty JAVA_HOME and no system paths, got nil")
	}
}

func TestFindCACerts_NotFound_Error(t *testing.T) {
	tmp := t.TempDir()
	javaHome := filepath.Join(tmp, "jdk")
	if err := os.MkdirAll(javaHome, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := FindCACerts(javaHome, optsNoSystem()); err == nil {
		t.Fatalf("expected error when cacerts does not exist")
	}
}

func Test_isFile_and_isSymlink(t *testing.T) {
	tmp := t.TempDir()

	f := filepath.Join(tmp, "f")
	mustWriteFile(t, f, []byte("x"))

	if !isFile(f) {
		t.Fatalf("isFile(%q) = false, want true", f)
	}

	link := filepath.Join(tmp, "l")
	mustSymlink(t, f, link)

	if !isSymlink(link) {
		t.Fatalf("isSymlink(%q) = false, want true", link)
	}
	// sanity: a regular file is not a symlink
	if isSymlink(f) {
		t.Fatalf("isSymlink(%q) = true, want false", f)
	}
}
