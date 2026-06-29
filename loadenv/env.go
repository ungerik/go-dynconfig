// Package loadenv provides configuration loaders that overlay environment
// variables on top of a parsed file, plus the configurable ParseEnv function
// they use.
//
// It is split into its own module so that the core github.com/ungerik/go-dynconfig
// module does not depend on github.com/caarlos0/env/v7. Import this package only
// when you need environment-variable overrides.
//
// The loaders returned here satisfy the load function signature expected by
// dynconfig.LoadAndWatch / MustLoadAndWatch:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    loadenv.LoadEnvJSON[*Config],
//	    nil, nil, nil, nil,
//	)
package loadenv

import (
	"reflect"

	env "github.com/caarlos0/env/v7"
)

// ParseEnv is a configurable function that parses environment variables into a struct.
//
// The default implementation uses github.com/caarlos0/env/v7 to parse environment
// variables based on `env` struct tags. You can replace this function with a custom
// implementation if needed.
//
// Struct Tag Format:
//   - env:"VAR_NAME" - Simple environment variable mapping
//   - env:"VAR_NAME,required" - Required variable (error if not set)
//   - env:"VAR_NAME" envDefault:"value" - Default value if not set
//   - env:"VAR_NAME,expand" - Expand ${OTHER_VAR} references
//
// Supported Types:
//   - All basic types: string, bool, int, int8, int16, int32, int64, uint, uint8, etc.
//   - float32, float64
//   - time.Duration (parsed from strings like "1h30m")
//   - url.URL (parsed from URL strings)
//   - Slices and arrays (comma-separated values)
//   - Maps (format: key1:value1,key2:value2)
//   - Custom types implementing encoding.TextUnmarshaler
//
// This function is used internally by LoadEnvJSON and LoadEnvXML.
//
// Example usage:
//
//	type Config struct {
//	    // Simple mapping
//	    Database string `env:"DB_NAME"`
//
//	    // With default value
//	    Port int `env:"PORT" envDefault:"8080"`
//
//	    // Required variable
//	    APIKey string `env:"API_KEY,required"`
//
//	    // Variable expansion
//	    LogPath string `env:"LOG_PATH,expand" envDefault:"${HOME}/logs"`
//
//	    // Comma-separated list
//	    Hosts []string `env:"ALLOWED_HOSTS" envSeparator:","`
//
//	    // Duration
//	    Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`
//	}
//
//	config := &Config{}
//	if err := loadenv.ParseEnv(config); err != nil {
//	    log.Fatal(err)
//	}
//
// Custom Parser Example:
//
//	// Replace with custom implementation
//	loadenv.ParseEnv = func(dest any) error {
//	    // Your custom environment parsing logic
//	    return customEnvParser(dest)
//	}
//
// The default implementation handles pointer-to-pointer dereferencing automatically,
// so it works correctly when called with **T as well as *T.
var ParseEnv = func(dest any) error {
	// Deref pointer to pointer because env.Parse
	// only accepts pointers to structs
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Pointer {
		dest = v.Elem().Interface()
	}
	return env.Parse(dest)
}
