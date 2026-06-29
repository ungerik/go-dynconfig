# go-dynconfig

Dynamic configuration loading with automatic file watching and reloading. Perfect for applications that need to update configuration without restart.

[![Go Reference](https://pkg.go.dev/badge/github.com/ungerik/go-dynconfig.svg)](https://pkg.go.dev/github.com/ungerik/go-dynconfig)
[![Go Report Card](https://goreportcard.com/badge/github.com/ungerik/go-dynconfig)](https://goreportcard.com/report/github.com/ungerik/go-dynconfig)

## Features

- **Automatic Reloading**: Watches config files and reloads on changes
- **Type-Safe**: Generic API ensures type safety at compile time
- **Multiple Formats**: Built-in support for JSON, XML, and text files
- **Environment Variables**: Merge environment variables with file-based config (via the `loadenv` submodule)
- **Error Recovery**: Configurable error handling with fallback values
- **Thread-Safe**: All operations are safe for concurrent use
- **Minimal Dependencies**: The core module only needs `ungerik/go-fs` and `golang.org/x/sys`; environment-variable support is isolated in the `github.com/ungerik/go-dynconfig/loadenv` submodule, which additionally uses `caarlos0/env/v7`

## Installation

```bash
go get github.com/ungerik/go-dynconfig
```

## Quick Start

### JSON Configuration

```go
package main

import (
    "log"
    "github.com/ungerik/go-dynconfig"
)

type Config struct {
    Host string `json:"host"`
    Port int    `json:"port"`
    Debug bool  `json:"debug"`
}

func main() {
    // Load and watch config.json
    config := dynconfig.MustLoadAndWatch(
        "config.json",
        dynconfig.LoadJSON[*Config],
        nil, // save (write-back function used by Set)
        nil, // onLoad callback
        nil, // onError (nil = panic on error)
        nil, // onInvalidate callback
    )

    // Get always returns the latest configuration
    cfg := config.Get()
    log.Printf("Running on %s:%d (debug=%v)", cfg.Host, cfg.Port, cfg.Debug)

    // Config automatically reloads when file changes
    // No restart needed!
}
```

### Text File (Line-Based)

```go
// Load email blacklist from text file
var emailBlacklist = dynconfig.MustLoadAndWatch(
    "email-blacklist.txt",
    dynconfig.LoadStringLineSetTrimSpace,
    nil, // save (write-back function used by Set)
    // onLoad: log successful loads
    func(loaded map[string]struct{}) map[string]struct{} {
        log.Printf("Loaded %d blacklisted emails", len(loaded))
        return loaded
    },
    // onError: provide default on error
    func(err error) map[string]struct{} {
        log.Printf("Error loading blacklist: %v", err)
        return map[string]struct{}{"spam@example.com": {}}
    },
    nil, // onInvalidate
)

func Blacklisted(email string) bool {
    blacklist := emailBlacklist.Get()
    _, exists := blacklist[email]
    return exists
}
```

## Table of Contents

- [Core Concepts](#core-concepts)
- [Configuration Formats](#configuration-formats)
  - [JSON](#json)
  - [XML](#xml)
  - [Text Files](#text-files)
- [Environment Variables](#environment-variables)
- [Callbacks](#callbacks)
- [Error Handling](#error-handling)
- [Advanced Usage](#advanced-usage)
- [Best Practices](#best-practices)

## Core Concepts

### Loader

The `Loader[T]` is the core type that manages configuration loading and watching:

```go
type Loader[T any] struct {
    // ... internal fields
}
```

Key characteristics:
- **Generic**: Type parameter `T` ensures type safety
- **Thread-Safe**: All methods can be called concurrently
- **Nil-Safe**: Methods can be called on nil loaders (returns zero value)
- **Automatic**: Watches file and reloads on changes

### Lifecycle

1. **Initial Load**: Configuration is loaded when `LoadAndWatch` is called
2. **Watching**: File system events trigger reload
3. **Invalidation**: File changes mark config as stale
4. **Reload**: Next `Get()` or `Load()` call reloads the config
5. **Callbacks**: Registered callbacks are invoked during lifecycle events

```
[Create] -> [Load] -> [Watch] -> [Change Detected] -> [Invalidate] -> [Reload]
                         ↑                                               |
                         └───────────────────────────────────────────────┘
```

## Configuration Formats

### JSON

#### Basic JSON Loading

```go
type AppConfig struct {
    ServerURL string   `json:"server_url"`
    Timeout   int      `json:"timeout"`
    Features  []string `json:"features"`
}

config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*AppConfig],
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

Example `config.json`:
```json
{
  "server_url": "https://api.example.com",
  "timeout": 30,
  "features": ["auth", "logging", "metrics"]
}
```

#### JSON with Environment Variables

Override JSON values with environment variables:

```go
type Config struct {
    DatabaseURL string `json:"database_url" env:"DATABASE_URL"`
    APIKey      string `json:"api_key" env:"API_KEY"`
    Port        int    `json:"port" env:"PORT"`
}

// Environment-variable overrides live in the separate loadenv submodule:
//   import "github.com/ungerik/go-dynconfig/loadenv"
//
// loadenv.LoadEnvJSON first loads JSON, then overrides with env vars
config := dynconfig.MustLoadAndWatch(
    "config.json",
    loadenv.LoadEnvJSON[*Config],
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

With `config.json`:
```json
{
  "database_url": "localhost:5432",
  "api_key": "default-key",
  "port": 8080
}
```

And environment:
```bash
export API_KEY="production-key"
export PORT=9000
```

Result: `api_key` and `port` are overridden by environment variables.

### XML

```go
type ServerConfig struct {
    XMLName xml.Name `xml:"server"`
    Host    string   `xml:"host"`
    Port    int      `xml:"port"`
}

config := dynconfig.MustLoadAndWatch(
    "config.xml",
    dynconfig.LoadXML[*ServerConfig],
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

Example `config.xml`:
```xml
<?xml version="1.0"?>
<server>
    <host>localhost</host>
    <port>8080</port>
</server>
```

#### XML with Environment Variables

```go
// loadenv.LoadEnvXML lives in the separate loadenv submodule:
//   import "github.com/ungerik/go-dynconfig/loadenv"
config := dynconfig.MustLoadAndWatch(
    "config.xml",
    loadenv.LoadEnvXML[*ServerConfig], // Merges env vars
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

### Text Files

#### Single String

```go
// Load entire file as a string
version := dynconfig.MustLoadAndWatch(
    "VERSION",
    dynconfig.LoadStringTrimSpace,
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)

fmt.Println("Version:", version.Get()) // Version: 1.2.3
```

#### String Slice (Lines)

```go
// Load file as slice of lines
allowedIPs := dynconfig.MustLoadAndWatch(
    "allowed-ips.txt",
    dynconfig.LoadStringLinesTrimSpace,
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)

for _, ip := range allowedIPs.Get() {
    fmt.Println("Allowed:", ip)
}
```

Example `allowed-ips.txt`:
```
192.168.1.1
10.0.0.1
172.16.0.1
```

#### String Set (Unique Lines)

```go
// Load file as set (map[string]struct{})
bannedWords := dynconfig.MustLoadAndWatch(
    "banned-words.txt",
    dynconfig.LoadStringLineSetTrimSpace,
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)

func Banned(word string) bool {
    _, exists := bannedWords.Get()[word]
    return exists
}
```

#### Custom String Types

Use type constraints for custom string types:

```go
type Email string
type EmailSet map[Email]struct{}

emailBlacklist := dynconfig.MustLoadAndWatch(
    "blacklist.txt",
    dynconfig.LoadStringLineSetTrimSpaceT[Email],
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)

blacklist := emailBlacklist.Get()
if _, blocked := blacklist[Email("test@example.com")]; blocked {
    log.Println("Email is blacklisted")
}
```

## Environment Variables

The `env` struct tag controls environment variable parsing:

```go
type Config struct {
    // Required env var
    APIKey string `env:"API_KEY,required"`

    // With default value
    LogLevel string `env:"LOG_LEVEL" envDefault:"info"`

    // Multiple possible env vars
    Host string `env:"HOST,HOSTNAME" envDefault:"localhost"`

    // Parse as specific type
    Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`

    // Comma-separated list
    AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
}
```

### Custom Environment Parser

Override the default parser:

```go
import "github.com/ungerik/go-dynconfig/loadenv"

func init() {
    // Use custom environment parser
    loadenv.ParseEnv = func(dest any) error {
        // Your custom parsing logic
        return myCustomParser(dest)
    }
}
```

## Callbacks

### onLoad Callback

Transform or validate loaded configuration:

```go
config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    // onLoad: validate and transform
    func(loaded *Config) *Config {
        // Validate
        if loaded.Port < 1 || loaded.Port > 65535 {
            loaded.Port = 8080 // Fix invalid port
        }

        // Transform
        loaded.Host = strings.ToLower(loaded.Host)

        log.Printf("Loaded config: %#v", loaded)
        return loaded
    },
    nil, nil,
)
```

### onError Callback

Handle errors and provide fallback configuration:

```go
defaultConfig := &Config{
    Host: "localhost",
    Port: 8080,
}

config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil,
    // onError: log and return default
    func(err error) *Config {
        log.Printf("Config error: %v (using default)", err)
        return defaultConfig
    },
    nil,
)
```

### onInvalidate Callback

React to configuration changes:

```go
config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil, nil,
    // onInvalidate: notify that config will be reloaded
    func() {
        log.Println("Config file changed, will reload on next access")
        // Could trigger cleanup, cache invalidation, etc.
    },
)
```

## Error Handling

### With Error Recovery

```go
config, err := dynconfig.LoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil,
    // onError provides fallback
    func(err error) *Config {
        return &Config{Host: "localhost", Port: 8080}
    },
    nil,
)
// err is nil even if initial load failed (fallback used)
```

### Without Error Recovery

```go
config, err := dynconfig.LoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil,
    nil, // No onError = return error
    nil,
)
if err != nil {
    log.Fatal(err) // Handle error
}
```

### Panic on Error

```go
// MustLoadAndWatch panics on error
config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

## Advanced Usage

### Custom Load Function

Implement your own loader:

```go
func loadCustomFormat(file fs.File) (*Config, error) {
    data, err := file.ReadAll()
    if err != nil {
        return nil, err
    }

    // Your custom parsing logic
    config := &Config{}
    // ... parse data into config ...

    return config, nil
}

config := dynconfig.MustLoadAndWatch(
    "config.custom",
    loadCustomFormat,
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

### Manual Control

Fine-grained control over loading and watching:

```go
// Create without loading
loader := dynconfig.NewLoader(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)

// Start watching manually
if err := loader.Watch(); err != nil {
    log.Fatal(err)
}

// Load manually
config, err := loader.Load()
if err != nil {
    log.Fatal(err)
}

// Check if loaded
if loader.Loaded() {
    fmt.Println("Config is loaded")
}

// Force reload
loader.Invalidate()
config = loader.Get() // Triggers reload

// Stop watching
if err := loader.Unwatch(); err != nil {
    log.Fatal(err)
}
```

### Atomic Mutation (`Mutate` and `Set`)

Two methods write the configuration file back:

- **`Mutate(reload, fn)`** does a read-modify-write: it hands your callback the
  current configuration and writes back what the callback returns. Use it when
  the new value depends on the current contents (incrementing a counter, adding
  to a set). With `reload` false it reuses the cached config when one is valid;
  with `reload` true it always reads the file first, for when you can't rely on
  the cache reflecting the latest on-disk value.
- **`Set`** writes a complete value you already hold, replacing the file's
  contents. It does not read or call back.

For files on the local file system both acquire an **exclusive
operating-system lock** (`flock`) on the file's parent directory and hold it for
the whole write
cycle. The new content is written to a temporary file in the same directory and
then **atomically renamed** over the target, so a reader (or another process)
never observes a half-written file and a crash leaves the original intact. A
second process that writes a file in that directory through `Mutate` or `Set`
blocks until the first one finishes, so concurrent processes cannot interleave
their writes. Within the process the loader's mutex additionally serializes both
against `Load`, `Get`, and `Invalidate`.

The write-back function is passed to the constructor right after the `load`
function; the package ships `SaveJSON[T]` and `SaveXML[T]` as factories that
return the write counterparts to `LoadJSON` and `LoadXML` (pass optional indent
strings, e.g. `SaveJSON[Counter]("  ")`). `Mutate` reuses the cached
configuration when one is valid and only re-reads from disk when the cache is
empty or has been invalidated (the same caching `Get` and `Load` use); `Set`
never reads.

```go
type Counter struct {
    Value int `json:"value"`
}

loader := dynconfig.MustLoadAndWatch(
    "counter.json",
    dynconfig.LoadJSON[Counter],
    dynconfig.SaveJSON[Counter]("  "),
    nil, nil, nil,
)

// Read-modify-write: increment the counter (reload=false uses the cache)
err := loader.Mutate(false, func(c Counter) (Counter, error) {
    c.Value++
    return c, nil
})
if err != nil {
    log.Fatal(err)
}

// Direct write: replace the whole value
err = loader.Set(Counter{Value: 0})
if err != nil {
    log.Fatal(err)
}
```

The `Mutate` callback receives the current configuration by value and returns the
new value to write. If it returns an error, nothing is written. A failed save is
likewise discarded before the rename, so the file is never left partially
written. On success the loader's cache is updated to the new value, and—if the
file is being watched—the write also triggers the normal invalidation.

Neither `Mutate` nor `Set` applies the `onLoad` callback. `Mutate` assumes
`onLoad` does not modify the value, so the value handed to the callback is the
same one `Get` returns and the result is cached as-is. If you only use `onLoad`
to log loads, that logging is unnecessary here: do it inside the `Mutate`
callback.

Any format works by supplying a custom save function to the constructor:

```go
loader := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[Config],
    func(f fs.File, c Config) error {
        return f.WriteJSON(context.Background(), c)
    },
    nil, nil, nil,
)
```

#### How-to: atomic updates in practice

A common use is maintaining a set (allowlist, blocklist, feature flags) stored as
a line-separated text file. `LoadStringLineSet` / `SaveStringLineSet` map the file
to a `map[string]struct{}`, so a mutation is just adding and removing keys:

```go
loader := dynconfig.NewLoader(
    "allowlist.txt",
    dynconfig.LoadStringLineSet,
    dynconfig.SaveStringLineSet(),
    nil, nil, nil,
)

err := loader.Mutate(false, func(set map[string]struct{}) (map[string]struct{}, error) {
    set["new.example.com"] = struct{}{} // add a member
    delete(set, "old.example.com")      // remove a member
    return set, nil
})
```

**Gotchas**

- **`Mutate` needs an existing file; `Set` can create one.** `Mutate` reads
  before it mutates, so a missing file returns a read error. `Set` does not read,
  so it creates the file if absent (the parent directory must exist).
- **The `Mutate` callback must return the new value.** It receives the config by
  value and returns the value to write; returning a non-nil error writes nothing.
- **The `Mutate` callback runs under the lock.** Keep it a fast, in-memory
  transform. Slow work inside it (network calls, disk I/O, blocking) holds the
  directory lock the whole time, stalling other processes' `Mutate`/`Set` on any
  file in that directory and every in-process loader call. Do expensive work
  before calling `Mutate`.
- **`reload` controls cache vs fresh read.** `Mutate(false, ...)` reuses the
  cached config when valid, so to be sure it sees another process's write either
  pass `reload` true, run a watcher (which invalidates the cache when the file
  changes), or call `Invalidate()` first. `Mutate(true, ...)` always reads the
  file first (under the same lock), so it never acts on a stale cache.
- **Same-directory calls serialize.** The lock is on the parent directory, so
  `Mutate`/`Set` on two different files in the same directory block one on the
  other. Keep independently updated configs in separate directories if that
  matters.
- **Non-local or non-Unix is best-effort.** On remote or virtual go-fs file
  systems, or platforms without `flock`, both fall back to an in-place overwrite
  with no OS lock or atomic rename: safe within the process (mutex), not across
  processes.

#### Explanation: why a directory lock and an atomic rename

Three things can go wrong when several processes update the same config file:

- **Lost updates** — two processes read value `5`, both write `6`, one increment vanishes.
- **Torn reads** — a reader sees a half-written file mid-save and fails to parse it.
- **Crash corruption** — a process dies mid-write, leaving a truncated file.

On a local file system the lock and rename close all three:

- An exclusive `flock` held for the whole write cycle serializes writers, so a
  `Mutate` that reads fresh and a concurrent writer can't lose each other's
  updates. (Because `Mutate` may reuse a cached value, lost-update avoidance also
  needs the cache to be fresh, hence the watcher / `Invalidate()` note above.)
- Writing to a temp file and `rename`-ing it over the target is atomic, so a
  reader always sees either the old file or the new one, never a partial one, and
  a crash leaves the original intact.

The lock is on the **parent directory**, not the file, because the atomic rename
replaces the file's inode. A lock held on the old inode would stop excluding a
process that opened the new one, and lost updates would creep back in. The
directory inode is stable, so it is a sound lock target. The cost is that
`Mutate`/`Set` calls on different files in the same directory serialize against
each other.

| Guarantee          | Local + Unix    | Other (fallback)  |
| ------------------ | --------------- | ----------------- |
| Serialized writes  | yes             | in-process only   |
| No torn reads      | yes             | no                |
| Crash-safe write   | yes             | no                |

The lock is **advisory**: it only excludes other processes that also go through
`Mutate`/`Set`. A writer that ignores the lock, or edits the file by hand, is not
blocked.

### Configuration Composition

Combine multiple config files:

```go
type DatabaseConfig struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

type CacheConfig struct {
    TTL      int  `json:"ttl"`
    MaxSize  int  `json:"max_size"`
}

type AppConfig struct {
    Database DatabaseConfig
    Cache    CacheConfig
}

func loadComposedConfig(file fs.File) (*AppConfig, error) {
    // Load main config
    mainConfig, err := dynconfig.LoadJSON[*AppConfig](file)
    if err != nil {
        return nil, err
    }

    // Load database config from separate file
    dbConfig, _ := dynconfig.LoadJSON[*DatabaseConfig]("database.json")
    if dbConfig != nil {
        mainConfig.Database = *dbConfig
    }

    return mainConfig, nil
}

config := dynconfig.MustLoadAndWatch(
    "app.json",
    loadComposedConfig,
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

### Configuration Validation

```go
type Config struct {
    Port    int      `json:"port"`
    Hosts   []string `json:"hosts"`
}

func (c *Config) Validate() error {
    if c.Port < 1 || c.Port > 65535 {
        return fmt.Errorf("invalid port: %d", c.Port)
    }
    if len(c.Hosts) == 0 {
        return fmt.Errorf("at least one host required")
    }
    return nil
}

config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    // Validate in onLoad
    func(loaded *Config) *Config {
        if err := loaded.Validate(); err != nil {
            log.Fatalf("Invalid config: %v", err)
        }
        return loaded
    },
    nil, nil,
)
```

### Hot Reload Notification

Notify application components when config changes:

```go
type ConfigObserver interface {
    OnConfigChange(*Config)
}

var observers []ConfigObserver

config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    // Notify observers on load
    func(loaded *Config) *Config {
        for _, observer := range observers {
            observer.OnConfigChange(loaded)
        }
        return loaded
    },
    nil,
    // Prepare for reload
    func() {
        log.Println("Config invalidated, reload pending")
    },
)
```

## Best Practices

### 1. Use Callbacks for Logging

```go
// ✅ Good: Log config changes
config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    func(loaded *Config) *Config {
        log.Printf("Config loaded: %+v", loaded)
        return loaded
    },
    func(err error) *Config {
        log.Printf("Config load failed: %v", err)
        return defaultConfig
    },
    func() {
        log.Println("Config file changed")
    },
)
```

### 2. Provide Sensible Defaults

```go
// ✅ Good: Always have a fallback
var defaultConfig = &Config{
    Host:    "localhost",
    Port:    8080,
    Timeout: 30 * time.Second,
}

config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    nil,
    func(err error) *Config {
        return defaultConfig
    },
    nil,
)
```

### 3. Validate Configuration

```go
// ✅ Good: Validate early
config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadJSON[*Config],
    func(loaded *Config) *Config {
        if err := validateConfig(loaded); err != nil {
            log.Fatalf("Invalid config: %v", err)
        }
        return loaded
    },
    nil, nil,
)
```

### 4. Use Environment Variables for Secrets

```go
// ✅ Good: Secrets in env, config in file
type Config struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    // Secret comes from environment
    APIKey   string `env:"API_KEY,required"`
    DBPass   string `env:"DB_PASSWORD,required"`
}

config := dynconfig.MustLoadAndWatch(
    "config.json",
    loadenv.LoadEnvJSON[*Config], // Merges env vars (from the loadenv submodule)
    nil, // save (write-back function used by Set)
    nil, nil, nil,
)
```

## API Reference

### Core Types

- `Loader[T]` - Main configuration loader with file watching
- `LoadAndWatch[T](file, load, save, onLoad, onError, onInvalidate) (*Loader[T], error)` - Create and start loader
- `MustLoadAndWatch[T](...) *Loader[T]` - Like LoadAndWatch but panics on error
- `NewLoader[T](...) *Loader[T]` - Create loader without loading

### Loader Methods

- `Get() T` - Get current config (reloads if needed)
- `Load() (T, error)` - Load config and return any error
- `Loaded() bool` - Check if config is loaded
- `Invalidate()` - Mark config as needing reload
- `Mutate(reload bool, mutate func(T) (T, error)) error` - Read-modify-write under an exclusive directory lock with an atomic rename (local files); with `reload` false uses the cached config when valid, with `reload` true always reads fresh from disk first; the `save` function is passed to the constructor
- `Set(config T) error` - Write a complete config value directly under the same lock and atomic rename (no read, no callback)
- `Watch() error` - Start watching file
- `Unwatch() error` - Stop watching file
- `File() fs.File` - Get watched file path

### JSON Loaders

- `LoadJSON[T](file) (T, error)` - Load JSON file
- `SaveJSON[T](indent ...string) func(file, config) error` - Returns a JSON write-back function (counterpart to LoadJSON)

### XML Loaders

- `LoadXML[T](file) (T, error)` - Load XML file
- `SaveXML[T](indent ...string) func(file, config) error` - Returns an XML write-back function (counterpart to LoadXML)

### Text Loaders

- `LoadString(file) (string, error)` - Load as string
- `LoadStringTrimSpace(file) (string, error)` - Load and trim
- `LoadStringLines(file) ([]string, error)` - Load as line slice
- `LoadStringLinesTrimSpace(file) ([]string, error)` - Load lines, trim each
- `LoadStringLineSet(file) (map[string]struct{}, error)` - Load as line set
- `LoadStringLineSetTrimSpace(file) (map[string]struct{}, error)` - Load set, trim lines

All text loaders have generic `T` variants (e.g., `LoadStringT[T]`, `LoadStringLinesT[T]`) for custom string types.

### Environment Variables (`loadenv` submodule)

Environment-variable support lives in the separate module
`github.com/ungerik/go-dynconfig/loadenv`, so the core module does not depend on
`caarlos0/env/v7`. Import it only when you need env overrides:

```go
import "github.com/ungerik/go-dynconfig/loadenv"
```

- `loadenv.LoadEnvJSON[T](file) (T, error)` - Load JSON and merge env vars
- `loadenv.LoadEnvXML[T](file) (T, error)` - Load XML and merge env vars
- `loadenv.ParseEnv(dest any) error` - Parse env vars into struct (customizable)

## Examples

See the [example](example/) directory for complete working examples.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See [LICENSE](LICENSE) file for details.

## Related Projects

- [go-fs](https://github.com/ungerik/go-fs) - File system abstraction (used internally)
