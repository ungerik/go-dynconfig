package dynconfig

import (
	"testing"

	"github.com/ungerik/go-fs"
)

// memFile creates an in-memory file system holding a single file with the given
// name and content, and returns an fs.File referencing it. The file system is
// closed automatically when the test finishes.
func memFile(t *testing.T, name, content string) fs.File {
	t.Helper()
	memFS, file, err := fs.NewSingleMemFileSystem(fs.NewMemFile(name, []byte(content)))
	if err != nil {
		t.Fatalf("NewSingleMemFileSystem: %s", err)
	}
	t.Cleanup(func() { memFS.Close() })
	return file
}

// missingMemFile returns an fs.File that points into an empty in-memory file
// system, so the referenced file does not exist. Useful for testing the
// error path of load functions. The file system is closed automatically.
func missingMemFile(t *testing.T, name string) fs.File {
	t.Helper()
	memFS, err := fs.NewMemFileSystem("/")
	if err != nil {
		t.Fatalf("NewMemFileSystem: %s", err)
	}
	t.Cleanup(func() { memFS.Close() })
	return memFS.JoinCleanFile("/", name)
}
