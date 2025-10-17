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
func NewLoader[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) *Loader[T] {
	return &Loader[T]{
		file:         file,
		load:         load,
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
//	    nil, nil, nil, // No callbacks
//	)
//	if err != nil {
//	    log.Fatal(err) // Initial load failed
//	}
func LoadAndWatch[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) (*Loader[T], error) {
	if load == nil {
		return nil, errors.New("load function must not be nil")
	}
	if file == "" {
		return nil, errors.New("file path must not be empty")
	}
	l := NewLoader(file, load, onLoad, onError, onInvalidate)
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
//	    nil, nil, nil,
//	)
//
//	// Safe to use - initial load succeeded or we would have panicked
//	config := loader.Get()
//	fmt.Println("API Key:", config.APIKey)
//
// See LoadAndWatch for more details.
func MustLoadAndWatch[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) *Loader[T] {
	l, err := LoadAndWatch(file, load, onLoad, onError, onInvalidate)
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
