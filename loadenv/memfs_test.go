package loadenv

import (
	"testing"

	"github.com/ungerik/go-fs"
)

// memFile creates an in-memory file system holding a single file with the given
// name and content, and returns an fs.File referencing it. The file system is
// closed, which unregisters it, when the test finishes.
func memFile(t *testing.T, name, content string) fs.File {
	t.Helper()
	memFS, err := fs.NewMemFileSystem(nil, fs.NewMemFile(name, []byte(content)))
	if err != nil {
		t.Fatalf("NewMemFileSystem: %s", err)
	}
	t.Cleanup(func() { memFS.Close() })
	err = fs.Register(memFS)
	if err != nil {
		t.Fatalf("Register: %s", err)
	}
	return memFS.RootDir().Join(name)
}
