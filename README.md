# go-dynconfig

Dynamic configuration loading with automatic file watching and reloading. Perfect for applications that need to update configuration without restart.

[![Go Reference](https://pkg.go.dev/badge/github.com/ungerik/go-dynconfig.svg)](https://pkg.go.dev/github.com/ungerik/go-dynconfig)
[![Go Report Card](https://goreportcard.com/badge/github.com/ungerik/go-dynconfig)](https://goreportcard.com/report/github.com/ungerik/go-dynconfig)

## Features

- **Automatic Reloading**: Watches config files and reloads on changes
- **Type-Safe**: Generic API ensures type safety at compile time
- **Multiple Formats**: Built-in support for JSON, XML, and text files
- **Environment Variables**: Merge environment variables with file-based config
- **Error Recovery**: Configurable error handling with fallback values
- **Thread-Safe**: All operations are safe for concurrent use
- **Zero Dependencies**: Uses standard library (except for env parsing)

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

// LoadEnvJSON first loads JSON, then overrides with env vars
config := dynconfig.MustLoadAndWatch(
    "config.json",
    dynconfig.LoadEnvJSON[*Config],
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
config := dynconfig.MustLoadAndWatch(
    "config.xml",
    dynconfig.LoadEnvXML[*ServerConfig], // Merges env vars
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
import "github.com/ungerik/go-dynconfig"

func init() {
    // Use custom environment parser
    dynconfig.ParseEnv = func(dest any) error {
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
    dynconfig.LoadEnvJSON[*Config], // Merges env vars
    nil, nil, nil,
)
```

## API Reference

### Core Types

- `Loader[T]` - Main configuration loader with file watching
- `LoadAndWatch[T](file, load, onLoad, onError, onInvalidate) (*Loader[T], error)` - Create and start loader
- `MustLoadAndWatch[T](...) *Loader[T]` - Like LoadAndWatch but panics on error
- `NewLoader[T](...) *Loader[T]` - Create loader without loading

### Loader Methods

- `Get() T` - Get current config (reloads if needed)
- `Load() (T, error)` - Load config and return any error
- `Loaded() bool` - Check if config is loaded
- `Invalidate()` - Mark config as needing reload
- `Watch() error` - Start watching file
- `Unwatch() error` - Stop watching file
- `File() fs.File` - Get watched file path

### JSON Loaders

- `LoadJSON[T](file) (T, error)` - Load JSON file
- `LoadEnvJSON[T](file) (T, error)` - Load JSON and merge env vars

### XML Loaders

- `LoadXML[T](file) (T, error)` - Load XML file
- `LoadEnvXML[T](file) (T, error)` - Load XML and merge env vars

### Text Loaders

- `LoadString(file) (string, error)` - Load as string
- `LoadStringTrimSpace(file) (string, error)` - Load and trim
- `LoadStringLines(file) ([]string, error)` - Load as line slice
- `LoadStringLinesTrimSpace(file) ([]string, error)` - Load lines, trim each
- `LoadStringLineSet(file) (map[string]struct{}, error)` - Load as line set
- `LoadStringLineSetTrimSpace(file) (map[string]struct{}, error)` - Load set, trim lines

All text loaders have generic `T` variants (e.g., `LoadStringT[T]`, `LoadStringLinesT[T]`) for custom string types.

### Environment Variables

- `ParseEnv(dest any) error` - Parse env vars into struct (customizable)

## Examples

See the [example](example/) directory for complete working examples.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

See [LICENSE](LICENSE) file for details.

## Related Projects

- [go-fs](https://github.com/ungerik/go-fs) - File system abstraction (used internally)
