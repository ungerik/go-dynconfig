package dynconfig

import (
	"testing"

	"github.com/ungerik/go-fs"
)

// newMemFS creates and registers an in-memory file system holding files. The
// file system is closed, which unregisters it, when the test finishes.
func newMemFS(t *testing.T, files ...fs.MemFile) *fs.MemFileSystem {
	t.Helper()
	memFS, err := fs.NewMemFileSystem(nil, files...)
	if err != nil {
		t.Fatalf("NewMemFileSystem: %s", err)
	}
	t.Cleanup(func() { memFS.Close() })
	err = fs.Register(memFS)
	if err != nil {
		t.Fatalf("Register: %s", err)
	}
	return memFS
}

// memFile creates an in-memory file system holding a single file with the given
// name and content, and returns an fs.File referencing it. The file system is
// closed automatically when the test finishes.
func memFile(t *testing.T, name, content string) fs.File {
	t.Helper()
	return newMemFS(t, fs.NewMemFile(name, []byte(content))).RootDir().Join(name)
}

// missingMemFile returns an fs.File that points into an empty in-memory file
// system, so the referenced file does not exist. Useful for testing the
// error path of load functions. The file system is closed automatically.
func missingMemFile(t *testing.T, name string) fs.File {
	t.Helper()
	return newMemFS(t).JoinCleanFile("/", name)
}
