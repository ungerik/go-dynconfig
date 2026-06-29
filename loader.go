// Package dynconfig provides dynamic configuration loading with automatic file watching
// and hot reload capabilities.
//
// The package offers type-safe configuration loading from various file formats (JSON, XML, text)
// with automatic reloading when files change, environment variable merging, and flexible
// error handling through callbacks.
//
// # Core Features
//
//   - Generic type-safe configuration loading
//   - Automatic file watching and hot reload
//   - Multiple format support (JSON, XML, text files)
//   - Environment variable integration
//   - Thread-safe operations
//   - Flexible error handling with callbacks
//   - Manual or automatic configuration lifecycle
//
// # Quick Start
//
// Load a JSON configuration file with automatic watching:
//
//	type Config struct {
//	    Host string `json:"host"`
//	    Port int    `json:"port"`
//	}
//
//	loader, err := dynconfig.LoadJSON[Config]("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	config := loader.Get() // Get current config
//	fmt.Println(config.Host, config.Port)
//
// Load configuration with environment variable merging:
//
//	loader, err := dynconfig.LoadEnvJSON[Config]("config.json", "APP")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Environment variables like APP_HOST and APP_PORT will override JSON values
//
// # Callbacks
//
// Use callbacks for custom behavior during the configuration lifecycle:
//
//	loader, err := dynconfig.LoadJSON[Config](
//	    "config.json",
//	    func(cfg Config) Config {
//	        // Called after successful load - transform config
//	        cfg.Host = strings.ToLower(cfg.Host)
//	        return cfg
//	    },
//	    func(err error) Config {
//	        // Called on load errors - provide fallback
//	        log.Printf("Config load error: %v", err)
//	        return Config{Host: "localhost", Port: 8080}
//	    },
//	    func() {
//	        // Called when config is invalidated (file changed)
//	        log.Println("Configuration file changed, reloading...")
//	    },
//	)
//
// # Error Handling Strategies
//
// 1. No recovery (fail on error):
//
//	loader, err := dynconfig.LoadJSON[Config]("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// 2. Fallback configuration:
//
//	loader, err := dynconfig.LoadJSON[Config](
//	    "config.json",
//	    nil,
//	    func(err error) Config {
//	        return Config{Host: "localhost", Port: 8080}
//	    },
//	)
//
// 3. Panic on error:
//
//	loader := dynconfig.MustLoadAndWatch(...)
//
// See the individual functions for more details and examples.
package dynconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ungerik/go-fs"
)

// Loader watches a file for changes and loads a configuration of type T from it
// using a load function. The configuration is automatically reloaded when the file changes.
//
// All methods are safe to call on a nil Loader and are thread-safe.
//
// The Loader uses a file system watcher to monitor the configuration file's directory.
// When the file is created or modified, the configuration is invalidated and reloaded
// on the next Get() or Load() call.
//
// Type Parameters:
//   - T: The configuration type to load from the file
//
// Example:
//
//	type AppConfig struct {
//	    Database string `json:"database"`
//	    Port     int    `json:"port"`
//	}
//
//	loader, err := dynconfig.LoadJSON[AppConfig]("config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get configuration (loads initially, then returns cached until invalidated)
//	config := loader.Get()
//	fmt.Printf("Running on port %d\n", config.Port)
//
//	// When config.json changes, next Get() will reload automatically
type Loader[T any] struct {
	mtx          sync.Mutex
	file         fs.File
	load         func(fs.File) (T, error)
	save         func(fs.File, T) error
	onLoad       func(T) T
	onError      func(error) T
	onInvalidate func()
	unwatch      func() error
	config       T
	loaded       bool
}

// NewLoader returns a new Loader for the type T without loading the configuration yet.
//
// This is useful when you need to set up the loader but delay the initial load,
// or when you want to manually control the loading process.
//
// Parameters:
//   - file: The file to load configuration from
//   - load: Function to load configuration from the file
//   - save: Optional function to write configuration back to the file, used by Set (can be nil)
//   - onLoad: Optional callback called after successful load (can be nil)
//   - onError: Optional callback to handle errors (can be nil)
//   - onInvalidate: Optional callback called when config is invalidated (can be nil)
//
// Example:
//
//	loader := dynconfig.NewLoader(
//	    "config.json",
//	    func(f fs.File) (Config, error) {
//	        var cfg Config
//	        err := f.ReadJSON(&cfg)
//	        return cfg, err
//	    },
//	    nil, // No save function
//	    nil, // No onLoad callback
//	    nil, // No error handling
//	    func() {
//	        log.Println("Config invalidated")
//	    },
//	)
//
//	// Start watching manually
//	if err := loader.Watch(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Load manually when needed
//	config, err := loader.Load()
//
// See LoadAndWatch for automatic loading and watching.
func NewLoader[T any](
	file fs.File,
	load func(fs.File) (T, error),
	save func(fs.File, T) error,
	onLoad func(T) T,
	onError func(error) T,
	onInvalidate func(),
) *Loader[T] {
	return &Loader[T]{
		file:         file,
		load:         load,
		save:         save,
		onLoad:       onLoad,
		onError:      onError,
		onInvalidate: onInvalidate,
	}
}

// LoadAndWatch creates a new Loader that immediately loads the configuration
// and starts watching the file for changes.
//
// This is the recommended way to create a Loader for most use cases.
// All Loader methods are thread-safe and can be called on a nil Loader.
//
// Parameters:
//   - file: The file to load configuration from
//   - load: Function to load configuration from the file (required, must not be nil)
//   - save: Optional function to write configuration back to the file, used by Set (can be nil)
//   - onLoad: Optional callback called after each successful load (can be nil)
//   - onError: Optional callback to handle load errors (can be nil)
//   - onInvalidate: Optional callback called when config is invalidated due to file changes (can be nil)
//
// File Watching:
//   - The file's directory is watched for file creation and modification events
//   - If the file doesn't exist yet but its directory does, watching starts successfully
//   - The configuration will be loaded when the file is created
//   - Returns an error if the directory cannot be watched
//
// Error Handling:
//   - If the initial load fails and onError is nil, returns the error
//   - If onError is provided, it handles the error and LoadAndWatch succeeds
//   - Subsequent load errors are handled by onError or return the last known config
//
// Example with all callbacks:
//
//	type Config struct {
//	    Host string `json:"host"`
//	    Port int    `json:"port"`
//	}
//
//	loader, err := dynconfig.LoadAndWatch(
//	    "config.json",
//	    func(f fs.File) (Config, error) {
//	        var cfg Config
//	        return cfg, f.ReadJSON(&cfg)
//	    },
//	    nil, // No save function
//	    func(cfg Config) Config {
//	        // Transform loaded config
//	        log.Printf("Loaded config: %+v", cfg)
//	        return cfg
//	    },
//	    func(err error) Config {
//	        // Handle errors with fallback
//	        log.Printf("Config error: %v, using defaults", err)
//	        return Config{Host: "localhost", Port: 8080}
//	    },
//	    func() {
//	        // React to file changes
//	        log.Println("Config file changed, will reload on next Get()")
//	    },
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example with minimal error handling:
//
//	loader, err := dynconfig.LoadAndWatch(
//	    "config.json",
//	    func(f fs.File) (Config, error) {
//	        var cfg Config
//	        return cfg, f.ReadJSON(&cfg)
//	    },
//	    nil, nil, nil, nil, // No save or callbacks
//	)
//	if err != nil {
//	    log.Fatal(err) // Initial load failed
//	}
func LoadAndWatch[T any](
	file fs.File,
	load func(fs.File) (T, error),
	save func(fs.File, T) error,
	onLoad func(T) T,
	onError func(error) T,
	onInvalidate func(),
) (*Loader[T], error) {
	if load == nil {
		return nil, errors.New("load function must not be nil")
	}
	if file == "" {
		return nil, errors.New("file path must not be empty")
	}
	l := NewLoader(file, load, save, onLoad, onError, onInvalidate)
	err := l.Watch() // May invalidate before load which is OK
	if err != nil {
		return nil, err
	}
	_, err = l.Load()
	if err != nil && onError == nil {
		// Unwatch and return error if no onError
		return nil, errors.Join(err, l.unwatch())
	}
	// In case of an error, onError was called within Load
	return l, nil
}

// MustLoadAndWatch calls LoadAndWatch and panics if it returns an error.
//
// This is a convenience wrapper for cases where you want the application
// to fail fast if the initial configuration cannot be loaded.
//
// Note: This only panics on initial load errors when onError is nil.
// If onError is provided, this function will not panic even if the initial load fails.
//
// Parameters: Same as LoadAndWatch
//
// Example:
//
//	type Config struct {
//	    APIKey string `json:"api_key"`
//	}
//
//	// Panic if config.json cannot be loaded initially
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    func(f fs.File) (Config, error) {
//	        var cfg Config
//	        return cfg, f.ReadJSON(&cfg)
//	    },
//	    nil, nil, nil, nil,
//	)
//
//	// Safe to use - initial load succeeded or we would have panicked
//	config := loader.Get()
//	fmt.Println("API Key:", config.APIKey)
//
// See LoadAndWatch for more details.
func MustLoadAndWatch[T any](
	file fs.File,
	load func(fs.File) (T, error),
	save func(fs.File, T) error,
	onLoad func(T) T,
	onError func(error) T,
	onInvalidate func(),
) *Loader[T] {
	l, err := LoadAndWatch(
		file,
		load,
		save,
		onLoad,
		onError,
		onInvalidate,
	)
	if err != nil {
		panic(err)
	}
	return l
}

// File returns the file path that is being watched for changes.
//
// Returns fs.InvalidFile if called on a nil Loader.
//
// Example:
//
//	loader, _ := dynconfig.LoadJSON[Config]("config.json")
//	fmt.Println("Watching:", loader.File())
func (l *Loader[T]) File() fs.File {
	if l == nil {
		return fs.InvalidFile
	}
	return l.file
}

// Loaded returns true if the configuration has been successfully loaded.
//
// This returns false if:
//   - The Loader is nil
//   - No successful load has occurred yet
//   - The configuration has been invalidated due to file changes
//
// Thread-safe.
//
// Example:
//
//	loader, _ := dynconfig.LoadJSON[Config]("config.json")
//	if loader.Loaded() {
//	    config := loader.Get() // Won't trigger a reload
//	}
func (l *Loader[T]) Loaded() bool {
	if l == nil {
		return false
	}
	l.mtx.Lock()
	defer l.mtx.Unlock()

	return l.loaded
}

// Invalidate marks the configuration as not loaded, forcing a reload on the next Get() or Load() call.
//
// This method:
//   - Sets the loaded flag to false
//   - Calls the onInvalidate callback if provided
//   - Is called automatically when the watched file changes
//   - Can be called manually to force a reload
//
// Safe to call on a nil Loader (no-op).
// Thread-safe.
//
// Example:
//
//	loader, _ := dynconfig.LoadJSON[Config]("config.json")
//
//	// Manually invalidate to force reload
//	loader.Invalidate()
//
//	// Next Get() will reload from file
//	config := loader.Get()
func (l *Loader[T]) Invalidate() {
	if l == nil {
		return
	}
	l.mtx.Lock()
	l.loaded = false
	l.mtx.Unlock()

	if l.onInvalidate != nil {
		l.onInvalidate()
	}
}

// Watch starts watching the file's directory for changes to the configuration file.
//
// Behavior:
//   - Monitors the file's parent directory for file system events
//   - Automatically calls Invalidate() when the file is created or modified
//   - File deletion does NOT trigger invalidation (maintains last known config)
//   - File recreation DOES trigger invalidation
//
// Returns an error if:
//   - Called on a nil Loader
//   - The file is already being watched
//   - The directory cannot be watched (e.g., doesn't exist or permission denied)
//
// Thread-safe.
//
// Note: The file itself doesn't need to exist for watching to start,
// only its parent directory must exist.
//
// Example:
//
//	loader := dynconfig.NewLoader(...)
//	if err := loader.Watch(); err != nil {
//	    log.Fatal(err)
//	}
//	defer loader.Unwatch()
func (l *Loader[T]) Watch() error {
	if l == nil {
		return errors.New("<nil> Loader")
	}
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.unwatch != nil {
		return fmt.Errorf("config file already watched: %s", l.file)
	}
	unwatch, err := l.file.Dir().Watch(func(f fs.File, e fs.Event) {
		if f == l.file && (e.HasCreate() || e.HasWrite()) {
			l.Invalidate()
		}
	})
	if err != nil {
		return fmt.Errorf("watch config file error: %w", err)
	}
	l.unwatch = unwatch
	return nil
}

// Unwatch stops watching the file for changes.
//
// Returns an error if:
//   - Called on a nil Loader
//   - The file is not currently being watched
//
// Thread-safe.
//
// Example:
//
//	loader, _ := dynconfig.LoadJSON[Config]("config.json")
//	// Automatically watching from LoadJSON
//
//	// Stop watching when done
//	if err := loader.Unwatch(); err != nil {
//	    log.Printf("Unwatch error: %v", err)
//	}
func (l *Loader[T]) Unwatch() error {
	if l == nil {
		return errors.New("<nil> Loader")
	}
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.unwatch == nil {
		return fmt.Errorf("config file not watched: %s", l.file)
	}
	err := l.unwatch()
	l.unwatch = nil
	return err
}

// Load returns the current configuration, loading it from the file if necessary.
//
// Behavior:
//   - If already loaded and not invalidated, returns cached configuration
//   - If not loaded or invalidated, loads from file first
//   - On successful load, calls onLoad callback if provided
//   - On load error with onError callback, returns onError result and the error
//   - On load error without onError, returns last known config and the error
//
// This method is thread-safe and can be called on a nil Loader
// (returns zero value of T and an error).
//
// Returns:
//   - The configuration value
//   - An error if loading failed (nil on success or if onError handled it)
//
// Example:
//
//	loader, _ := dynconfig.LoadJSON[Config]("config.json")
//
//	// Load or get cached config
//	config, err := loader.Load()
//	if err != nil {
//	    log.Printf("Config load error: %v", err)
//	    // config contains last known or onError result
//	}
//
//	// File changes invalidate, causing next Load() to reload
//	config2, _ := loader.Load() // Reloads if file changed
func (l *Loader[T]) Load() (T, error) {
	if l == nil {
		return *new(T), errors.New("<nil> Loader")
	}
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.loaded {
		return l.config, nil
	}

	config, err := l.load(l.file)
	if err != nil {
		if l.onError != nil {
			return l.onError(err), err
		}
		return l.config, err // Return last known config
	}
	if l.onLoad != nil {
		l.config = l.onLoad(config)
	} else {
		l.config = config
	}
	l.loaded = true
	return l.config, nil
}

// Get returns the current configuration, loading it from the file if necessary.
//
// This is a convenience method that wraps Load() and discards the error.
// It behaves identically to Load() but only returns the configuration value.
//
// Use this when:
//   - You have an onError callback that provides a fallback configuration
//   - You want simpler code and the error is logged elsewhere
//   - You're okay with receiving the last known config on errors
//
// Thread-safe. Safe to call on a nil Loader (returns zero value of T).
//
// Example:
//
//	loader, _ := dynconfig.LoadJSON[Config](
//	    "config.json",
//	    nil,
//	    func(err error) Config {
//	        return Config{Host: "localhost", Port: 8080}
//	    },
//	    nil,
//	)
//
//	// Simple access - error handled by onError callback
//	config := loader.Get()
//	fmt.Printf("Server: %s:%d\n", config.Host, config.Port)
//
// For error handling, use Load() instead.
func (l *Loader[T]) Get() T {
	config, _ := l.Load()
	return config
}

// Mutate atomically reads, mutates, and writes back the configuration file as a
// single read-modify-write operation. Use Set instead when you already hold the
// complete value and don't need the current on-disk contents.
//
// If reload is false, Mutate reuses the cached configuration when one is valid
// (the same caching Get and Load use) and only reads from disk when the cache is
// empty or has been invalidated. If reload is true, Mutate always reads the
// current on-disk content first, ignoring the cache; use that when you cannot
// rely on the cache reflecting the latest file (for example when no watcher is
// running) and need a lost-update-free read-modify-write.
//
// For files on the local file system Mutate acquires an exclusive operating-system
// advisory lock (flock) on the file's parent directory and holds it for the
// whole read-modify-write cycle. The mutated value is written to a temporary
// file in the same directory and then atomically renamed over the target, so a
// reader (or another process) never observes a partially written file and a
// crash leaves the original file intact. While the lock is held, any other
// process that also writes a file in that directory through Mutate or Set blocks
// until this call completes, so concurrent processes cannot interleave their
// writes. Within the process the Loader's mutex additionally serializes Mutate
// against Load, Get, Set, and Invalidate.
//
// The lock is taken on the directory rather than the file itself because the
// atomic rename replaces the file's inode; a lock held on the old inode would
// stop excluding a process that opened the new one. A consequence is that Mutate
// and Set calls on different files in the same directory also serialize against
// each other.
//
// The sequence is:
//
//  1. For a local file, open the parent directory and acquire an exclusive
//     OS-level lock on it.
//  2. Obtain the current configuration: when reload is false and a valid cached
//     value is available it is used as-is; otherwise the on-disk content is
//     parsed with the Loader's load function.
//  3. Call mutate with that value and use its returned value as the mutated
//     configuration.
//  4. Write the mutated value to a temporary file using the save function
//     passed to the constructor, then atomically rename it over the target.
//  5. Cache the mutated value so the next Get or Load returns it without
//     re-reading the file.
//  6. Release the lock and close the directory.
//
// On failure the configuration file is left untouched: a read or mutate error
// aborts before anything is written, and a save error discards the temporary
// file before the rename, so the original file is never partially overwritten.
//
// Unlike Load and Get, Mutate does NOT apply the onLoad callback. It assumes
// onLoad does not modify the value, so the freshly parsed value handed to mutate
// is the same value Get returns, and the mutated value is written and cached
// as-is. If you only use onLoad to log loads, that logging is unnecessary here:
// do it inside mutate instead.
//
// If the file is being watched, writing it will additionally trigger the normal
// invalidation, causing a later Get or Load to reload from disk.
//
// Requirements and caveats:
//   - mutate must be non-nil and a save function must have been passed to the
//     constructor (NewLoader, LoadAndWatch or MustLoadAndWatch).
//   - mutate runs while the lock is held, so keep it a fast, pure in-memory
//     transform. Slow work inside it (network calls, disk I/O, blocking) holds
//     the directory lock for that whole time, blocking other processes' Mutate
//     and Set on any file in the same directory as well as every in-process
//     Loader operation. Do expensive work before calling Mutate.
//   - With reload false, Mutate reuses the cached configuration, so to be sure it
//     sees a write made by another process either pass reload true, run a watcher
//     (which invalidates the cache when the file changes), or call Invalidate
//     before Mutate. (Set is unaffected, as it does not read.)
//   - Atomic, cross-process-safe writes only apply to files on the local file
//     system. For remote or virtual go-fs file systems Mutate falls back to an
//     in-place overwrite with the save function, without an OS lock or atomic
//     rename, protected only by the Loader's in-process mutex (so it is not
//     safe against other processes mutating the same file).
//   - The lock is advisory: it only excludes other processes that cooperate by
//     locking the same directory (as Mutate and Set do). It does not protect
//     against writers that ignore the lock.
//   - Exclusive locking is implemented with flock on Unix. On platforms without
//     a supported implementation Mutate likewise falls back to an in-place
//     overwrite rather than failing, protected only by the Loader's in-process
//     mutex.
//
// Safe to call on a nil Loader (returns an error). Thread-safe.
//
// Example atomically incrementing a counter stored as JSON, even across processes:
//
//	type Counter struct {
//	    Value int `json:"value"`
//	}
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "counter.json",
//	    dynconfig.LoadJSON[Counter],
//	    dynconfig.SaveJSON[Counter](),
//	    nil, nil, nil,
//	)
//
//	err := loader.Mutate(false, func(c Counter) (Counter, error) {
//	    c.Value++
//	    return c, nil
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (l *Loader[T]) Mutate(reload bool, mutate func(config T) (T, error)) (err error) {
	if l == nil {
		return errors.New("<nil> Loader")
	}
	if mutate == nil {
		return errors.New("Mutate() mutate function must not be nil")
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.save == nil {
		return errors.New("Mutate() requires a save function passed to the constructor")
	}

	atomic, localPath, release, e := l.lockForWrite()
	if e != nil {
		return fmt.Errorf("Mutate() %w", e)
	}
	defer func() { err = errors.Join(err, release()) }()

	// Reuse the cached configuration when it is valid; read from disk when reload
	// is requested or the cache is empty or has been invalidated.
	config := l.config
	if reload || !l.loaded {
		config, e = l.load(l.file)
		if e != nil {
			return fmt.Errorf("Mutate() read error: %w", e)
		}
	}
	config, e = mutate(config)
	if e != nil {
		return fmt.Errorf("Mutate() mutate error: %w", e)
	}
	e = l.writeConfig(atomic, localPath, config)
	if e != nil {
		return fmt.Errorf("Mutate() save error: %w", e)
	}

	// Cache the mutated value directly so it is immediately visible without
	// re-reading the file. Mutate deliberately does NOT apply onLoad: the mutate
	// function is handed, and returns, exactly the value Get exposes, so there is
	// nothing for onLoad to transform (see the doc comment). A file watcher, if
	// active, will additionally invalidate after observing the write.
	l.config = config
	l.loaded = true
	return nil
}

// Set atomically writes config to the configuration file, replacing its entire
// contents, and updates the cache.
//
// Set is the direct-write counterpart to Mutate: use Set when you already hold
// the complete configuration value, and Mutate when the new value must be
// derived from the current on-disk contents. Set does not read the file and does
// not call a callback; it just persists the value passed to it.
//
// Set uses the same locking and atomic-write machinery as Mutate (see Mutate for
// the full description): on the local file system it holds an exclusive lock on
// the parent directory and writes through a temporary file and atomic rename, so
// it has the same cross-process and crash safety and serializes against Mutate,
// Set, Load, Get, and Invalidate. On non-local file systems or platforms without
// flock it falls back to an in-place overwrite protected only by the in-process
// mutex.
//
// Because Set does not read the file first, it can create a new file (the parent
// directory must exist). Like Mutate, it does not apply the onLoad callback; the
// value passed is written and cached as-is.
//
// A save function must have been passed to the constructor. Safe to call on a nil
// Loader (returns an error). Thread-safe.
//
// Example writing a whole configuration value:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    dynconfig.LoadJSON[Config],
//	    dynconfig.SaveJSON[Config]("  "),
//	    nil, nil, nil,
//	)
//
//	err := loader.Set(Config{Host: "localhost", Port: 9090})
//	if err != nil {
//	    log.Fatal(err)
//	}
func (l *Loader[T]) Set(config T) (err error) {
	if l == nil {
		return errors.New("<nil> Loader")
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.save == nil {
		return errors.New("Set() requires a save function passed to the constructor")
	}

	atomic, localPath, release, e := l.lockForWrite()
	if e != nil {
		return fmt.Errorf("Set() %w", e)
	}
	defer func() { err = errors.Join(err, release()) }()

	e = l.writeConfig(atomic, localPath, config)
	if e != nil {
		return fmt.Errorf("Set() save error: %w", e)
	}

	// Cache the written value directly so it is immediately visible without
	// re-reading the file. A file watcher, if active, will additionally
	// invalidate after observing the write.
	l.config = config
	l.loaded = true
	return nil
}

// lockForWrite acquires the exclusive write lock for the configuration file when
// it is on a lockable local file system. It reports whether the atomic write
// path applies and the local path (empty for non-local file systems), and
// returns a release function that unlocks and closes the lock handle (a no-op
// when no lock was taken). The caller must hold l.mtx.
func (l *Loader[T]) lockForWrite() (atomic bool, localPath string, release func() error, err error) {
	localPath = l.file.LocalPath()
	if localPath == "" || !fsLockSupported {
		// Non-local go-fs file system or a platform without flock support: no OS
		// lock, the caller is protected only by the in-process mutex.
		return false, localPath, func() error { return nil }, nil
	}
	// Lock the parent directory, whose inode the atomic rename never replaces.
	d, err := os.Open(filepath.Dir(localPath))
	if err != nil {
		return false, "", nil, fmt.Errorf("open directory for locking error: %w", err)
	}
	err = lockFileExclusive(d)
	if err != nil {
		return false, "", nil, errors.Join(
			fmt.Errorf("lock error: %w", err),
			d.Close(),
		)
	}
	return true, localPath, func() error {
		return errors.Join(unlockFile(d), d.Close())
	}, nil
}

// writeConfig persists config to the file: atomically through a temporary file
// and rename on a lockable local file system, or with a plain in-place overwrite
// otherwise. The caller must hold l.mtx and, on the atomic path, the directory
// lock from lockForWrite.
func (l *Loader[T]) writeConfig(atomic bool, localPath string, config T) error {
	if atomic {
		return l.saveAtomic(localPath, config)
	}
	return l.save(l.file, config)
}

// saveAtomic writes config to a temporary file in the same directory as
// localPath using the Loader's save function, then atomically renames it over
// the target. A reader or another writer therefore never observes a partially
// written file, and a save error or crash leaves the original file intact.
// The caller must hold the directory lock.
func (l *Loader[T]) saveAtomic(localPath string, config T) (err error) {
	dir, name := filepath.Split(localPath)
	tmp, err := os.CreateTemp(dir, "."+name+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Close our handle immediately; the save function reopens the path to write.
	err = tmp.Close()
	if err != nil {
		return errors.Join(err, os.Remove(tmpPath))
	}
	// Remove the temp file unless the rename below consumes it.
	renamed := false
	defer func() {
		if !renamed {
			err = errors.Join(err, os.Remove(tmpPath))
		}
	}()

	err = l.save(fs.File(tmpPath), config)
	if err != nil {
		return err
	}
	// Preserve the original file's permission bits; os.CreateTemp uses 0600.
	if info, statErr := os.Stat(localPath); statErr == nil {
		err = os.Chmod(tmpPath, info.Mode().Perm())
		if err != nil {
			return err
		}
	}
	err = os.Rename(tmpPath, localPath)
	if err != nil {
		return err
	}
	renamed = true
	return nil
}
