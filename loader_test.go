package dynconfig

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ungerik/go-fs"
)

type counter struct {
	Value int `json:"value"`
}

func writeTempJSON(t *testing.T, name string, content string) fs.File {
	t.Helper()
	file := fs.File(filepath.Join(t.TempDir(), name))
	err := file.WriteAllString(content)
	if err != nil {
		t.Fatalf("write temp file: %s", err)
	}
	return file
}

// assertOnlyFile fails if dir contains anything other than the named file,
// catching temporary files left behind by the atomic save.
func assertOnlyFile(t *testing.T, dir, name string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %s", err)
	}
	for _, e := range entries {
		if e.Name() != name {
			t.Errorf("unexpected leftover file in directory: %s", e.Name())
		}
	}
}

func TestMutate_ReadModifyWrite(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)

	loader, err := LoadAndWatch(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)
	if err != nil {
		t.Fatalf("LoadAndWatch: %s", err)
	}
	defer loader.Unwatch() //nolint:errcheck

	err = loader.Mutate(false, func(c counter) (counter, error) {
		c.Value += 41
		return c, nil
	})
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}

	// Cache must reflect the mutated value immediately.
	if got := loader.Get().Value; got != 42 {
		t.Errorf("cached value = %d, want 42", got)
	}

	// The file on disk must contain the mutated value.
	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 42 {
		t.Errorf("on-disk value = %d, want 42", onDisk.Value)
	}
}

// TestMutate_UsesCachedConfigWhenValid verifies Mutate mutates the cached value
// (not the on-disk value) when the cache is valid, avoiding a disk read.
func TestMutate_UsesCachedConfigWhenValid(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)

	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	// Prime the cache with the initial value (loaded == true).
	if got := loader.Get().Value; got != 1 {
		t.Fatalf("initial value = %d, want 1", got)
	}

	// Change the file behind the loader's back (no watcher, cache stays valid).
	err := file.WriteAllString(`{"value": 100}`)
	if err != nil {
		t.Fatalf("overwrite file: %s", err)
	}

	// Mutate must use the cached 1 (not the on-disk 100) and write 2.
	err = loader.Mutate(false, func(c counter) (counter, error) {
		c.Value++
		return c, nil
	})
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}

	if got := loader.Get().Value; got != 2 {
		t.Errorf("cached value = %d, want 2 (cached 1 + 1)", got)
	}
	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 2 {
		t.Errorf("on-disk value = %d, want 2", onDisk.Value)
	}
}

// TestMutate_ReloadsAfterInvalidate verifies that once the cache is invalidated
// Mutate reads the fresh on-disk content again.
func TestMutate_ReloadsAfterInvalidate(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)

	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)
	if got := loader.Get().Value; got != 1 { // prime cache
		t.Fatalf("initial value = %d, want 1", got)
	}

	err := file.WriteAllString(`{"value": 100}`)
	if err != nil {
		t.Fatalf("overwrite file: %s", err)
	}
	loader.Invalidate() // cache is now stale

	// With the cache invalidated, Mutate reloads the fresh 100 and writes 101.
	err = loader.Mutate(false, func(c counter) (counter, error) {
		c.Value++
		return c, nil
	})
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}

	if got := loader.Get().Value; got != 101 {
		t.Errorf("value = %d, want 101 (fresh disk 100 + 1)", got)
	}
}

func TestMutate_MutateErrorLeavesFileUnchanged(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 7}`)

	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	wantErr := errors.New("boom")
	err := loader.Mutate(false, func(c counter) (counter, error) {
		c.Value = 999
		return c, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}

	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 7 {
		t.Errorf("on-disk value = %d, want 7 (unchanged)", onDisk.Value)
	}
}

func TestMutate_NoSaveFuncSet(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)
	loader := NewLoader(file, LoadJSON[counter], nil, nil, nil, nil)

	err := loader.Mutate(false, func(c counter) (counter, error) { return c, nil })
	if err == nil {
		t.Error("expected error when no save function is set")
	}
}

func TestMutate_NilArguments(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)
	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	err := loader.Mutate(false, nil)
	if err == nil {
		t.Error("expected error for nil mutate")
	}

	var nilLoader *Loader[counter]
	err = nilLoader.Mutate(false, func(c counter) (counter, error) { return c, nil })
	if err == nil {
		t.Error("expected error for nil Loader")
	}
}

func TestMutate_DoesNotApplyOnLoad(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 10}`)

	loader := NewLoader(
		file,
		LoadJSON[counter],
		SaveJSON[counter](),
		func(c counter) counter { c.Value *= 2; return c }, // onLoad doubles
		nil,
		nil,
	)

	err := loader.Mutate(false, func(c counter) (counter, error) { c.Value++; return c, nil })
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}

	// Mutate does not apply onLoad: it loads the raw 10, writes 11, and caches
	// 11 as-is (not the doubled 22 that onLoad would produce).
	if got := loader.Get().Value; got != 11 {
		t.Errorf("cached value = %d, want 11 (Mutate must not apply onLoad)", got)
	}

	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 11 {
		t.Errorf("on-disk value = %d, want 11", onDisk.Value)
	}
}

// TestMutate_NonLocalFileSystem verifies that Mutate works on a non-local go-fs
// file system (which cannot be flock-ed) by proceeding without an OS lock instead
// of returning an error.
func TestMutate_NonLocalFileSystem(t *testing.T) {
	file := memFile(t, "counter.json", `{"value": 5}`)
	if file.LocalPath() != "" {
		t.Fatalf("memFile unexpectedly has a local path: %s", file.LocalPath())
	}

	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	err := loader.Mutate(false, func(c counter) (counter, error) {
		c.Value += 10
		return c, nil
	})
	if err != nil {
		t.Fatalf("Mutate on non-local file: %s", err)
	}

	if got := loader.Get().Value; got != 15 {
		t.Errorf("cached value = %d, want 15", got)
	}

	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 15 {
		t.Errorf("on-disk value = %d, want 15", onDisk.Value)
	}
}

// TestMutate_ConcurrentGoroutines exercises the lock path under contention within
// a single process: many goroutines each increment the counter once and the final
// value must equal the number of increments with no lost updates.
func TestMutate_ConcurrentGoroutines(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 0}`)
	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			err := loader.Mutate(false, func(c counter) (counter, error) { c.Value++; return c, nil })
			if err != nil {
				t.Errorf("Mutate: %s", err)
			}
		}()
	}
	wg.Wait()

	var onDisk counter
	err := file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != n {
		t.Errorf("on-disk value = %d, want %d (no lost updates)", onDisk.Value, n)
	}
}

// TestMutate_SaveErrorLeavesFileUnchanged verifies the atomic-write guarantee:
// when the save function writes partial content and then fails, that content
// lands in the discarded temp file, so the original file is left completely
// intact rather than truncated or partially overwritten.
func TestMutate_SaveErrorLeavesFileUnchanged(t *testing.T) {
	if !fsLockSupported {
		t.Skip("requires OS file locking for the atomic write path")
	}
	file := writeTempJSON(t, "counter.json", `{"value": 7}`)

	wantErr := errors.New("save boom")
	// Write garbage before failing; on the atomic path it goes to the temp file
	// and is discarded, so the original must survive unchanged.
	badSave := func(f fs.File, _ counter) error {
		_ = f.WriteAllString("partial garbage")
		return wantErr
	}
	loader := NewLoader(file, LoadJSON[counter], badSave, nil, nil, nil)

	err := loader.Mutate(false, func(c counter) (counter, error) {
		c.Value = 999
		return c, nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped %v", err, wantErr)
	}

	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 7 {
		t.Errorf("on-disk value = %d, want 7 (unchanged after save error)", onDisk.Value)
	}

	// The discarded temp file must not linger next to the target.
	assertOnlyFile(t, filepath.Dir(file.LocalPath()), "counter.json")
}

// TestMutate_PreservesFileMode verifies the atomic rename keeps the original
// file's permission bits rather than the 0600 of the temp file os.CreateTemp makes.
func TestMutate_PreservesFileMode(t *testing.T) {
	if !fsLockSupported {
		t.Skip("requires OS file locking for the atomic write path")
	}
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)
	localPath := file.LocalPath()
	if err := os.Chmod(localPath, 0o640); err != nil {
		t.Fatalf("Chmod: %s", err)
	}

	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)
	err := loader.Mutate(false, func(c counter) (counter, error) { c.Value++; return c, nil })
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}

	info, err := os.Stat(localPath)
	if err != nil {
		t.Fatalf("Stat: %s", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Errorf("mode = %#o, want 0640 (preserved across atomic rename)", got)
	}
	assertOnlyFile(t, filepath.Dir(localPath), "counter.json")
}

// TestMutate_MultiProcess exercises the actual cross-process directory lock (not
// the in-process mutex) by re-invoking the test binary as N child processes that
// each increment the shared on-disk counter once. Each child uses a fresh loader
// (empty cache), so it reads the latest value under the lock. With correct
// locking the final value equals N; a lost update would mean the lock failed to
// exclude another process.
func TestMutate_MultiProcess(t *testing.T) {
	if path := os.Getenv("DYNCONFIG_MUTATE_WORKER"); path != "" {
		// Child process: increment once and exit before the parent body runs.
		l := NewLoader(fs.File(path), LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)
		if err := l.Mutate(false, func(c counter) (counter, error) { c.Value++; return c, nil }); err != nil {
			fmt.Fprintln(os.Stderr, "worker Mutate:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if !fsLockSupported {
		t.Skip("requires OS file locking for the atomic write path")
	}

	file := writeTempJSON(t, "counter.json", `{"value": 0}`)
	localPath := file.LocalPath()

	const n = 15
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			cmd := exec.Command(os.Args[0], "-test.run=^TestMutate_MultiProcess$")
			cmd.Env = append(os.Environ(), "DYNCONFIG_MUTATE_WORKER="+localPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("worker process failed: %v\n%s", err, out)
			}
		}()
	}
	wg.Wait()

	var onDisk counter
	err := file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != n {
		t.Errorf("on-disk value = %d, want %d (lost updates => directory lock not working cross-process)", onDisk.Value, n)
	}
	assertOnlyFile(t, filepath.Dir(localPath), "counter.json")
}

// TestMutate_ReloadReadsFreshContent verifies that Mutate with reload=true
// ignores a valid cache and reads the current on-disk value (where reload=false
// would reuse the cached value).
func TestMutate_ReloadReadsFreshContent(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)

	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	// Prime the cache with the initial value (loaded == true).
	if got := loader.Get().Value; got != 1 {
		t.Fatalf("initial value = %d, want 1", got)
	}

	// Change the file behind the loader's back (no watcher, cache stays valid).
	err := file.WriteAllString(`{"value": 100}`)
	if err != nil {
		t.Fatalf("overwrite file: %s", err)
	}

	// reload=true must read the fresh 100 (not the cached 1) and write 101.
	err = loader.Mutate(true, func(c counter) (counter, error) {
		c.Value++
		return c, nil
	})
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}

	if got := loader.Get().Value; got != 101 {
		t.Errorf("value = %d, want 101 (fresh disk 100 + 1)", got)
	}
	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 101 {
		t.Errorf("on-disk value = %d, want 101", onDisk.Value)
	}
}

// TestSet_WritesConfigDirectly verifies Set persists the given value verbatim,
// without reading or deriving from the current on-disk contents.
func TestSet_WritesConfigDirectly(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)
	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	err := loader.Set(counter{Value: 99})
	if err != nil {
		t.Fatalf("Set: %s", err)
	}

	if got := loader.Get().Value; got != 99 {
		t.Errorf("cached value = %d, want 99", got)
	}
	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 99 {
		t.Errorf("on-disk value = %d, want 99 (written directly, not derived from 1)", onDisk.Value)
	}
}

// TestSet_CreatesMissingFile verifies Set can create a not-yet-existing file
// (it does not read first), as long as the parent directory exists.
func TestSet_CreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	file := fs.File(filepath.Join(dir, "new.json")) // not created yet
	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	err := loader.Set(counter{Value: 5})
	if err != nil {
		t.Fatalf("Set: %s", err)
	}

	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 5 {
		t.Errorf("on-disk value = %d, want 5", onDisk.Value)
	}
	assertOnlyFile(t, dir, "new.json")
}

func TestSet_NoSaveFuncSet(t *testing.T) {
	file := writeTempJSON(t, "counter.json", `{"value": 1}`)
	loader := NewLoader(file, LoadJSON[counter], nil, nil, nil, nil)

	err := loader.Set(counter{Value: 2})
	if err == nil {
		t.Error("expected error when no save function is set")
	}
}

func TestSet_NilLoader(t *testing.T) {
	var nilLoader *Loader[counter]
	err := nilLoader.Set(counter{Value: 1})
	if err == nil {
		t.Error("expected error for nil Loader")
	}
}

// TestSet_NonLocalFileSystem verifies that Set works on a non-local go-fs file
// system by proceeding without an OS lock instead of returning an error.
func TestSet_NonLocalFileSystem(t *testing.T) {
	file := memFile(t, "counter.json", `{"value": 5}`)
	loader := NewLoader(file, LoadJSON[counter], SaveJSON[counter](), nil, nil, nil)

	err := loader.Set(counter{Value: 42})
	if err != nil {
		t.Fatalf("Set on non-local file: %s", err)
	}

	if got := loader.Get().Value; got != 42 {
		t.Errorf("cached value = %d, want 42", got)
	}
	var onDisk counter
	err = file.ReadJSON(context.Background(), &onDisk)
	if err != nil {
		t.Fatalf("ReadJSON: %s", err)
	}
	if onDisk.Value != 42 {
		t.Errorf("on-disk value = %d, want 42", onDisk.Value)
	}
}

// ExampleLoader_Mutate demonstrates an atomic read-modify-write of a string set
// stored as a line-separated text file. The configuration type is
// map[string]struct{}, loaded and saved with the package's LoadStringLineSet
// and SaveStringLineSet (which writes the members sorted, one per line).
func ExampleLoader_Mutate() {
	dir, err := os.MkdirTemp("", "dynconfig-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := fs.File(filepath.Join(dir, "tags.txt"))
	err = file.WriteAllString("alpha\nbeta")
	if err != nil {
		log.Fatal(err)
	}

	loader := NewLoader(
		file,
		LoadStringLineSet,   // text file -> map[string]struct{}
		SaveStringLineSet(), // map[string]struct{} -> text file, sorted
		nil, nil, nil,
	)

	// Atomically add "gamma" and remove "alpha".
	err = loader.Mutate(false, func(set map[string]struct{}) (map[string]struct{}, error) {
		set["gamma"] = struct{}{}
		delete(set, "alpha")
		return set, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	content, err := file.ReadAllString()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(content)

	// Output:
	// beta
	// gamma
}
