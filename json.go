package dynconfig

import (
	"context"
	"strings"

	"github.com/ungerik/go-fs"
)

// LoadJSON returns a function that loads JSON configuration from a file.
//
// This is a loader function compatible with LoadAndWatch and MustLoadAndWatch.
// The returned function unmarshals the JSON file into a configuration struct of type T.
//
// Type Parameters:
//   - T: The configuration type to unmarshal from JSON
//
// Example with LoadAndWatch:
//
//	type Config struct {
//	    Database string `json:"database"`
//	    Port     int    `json:"port"`
//	}
//
//	loader, err := dynconfig.LoadAndWatch(
//	    "config.json",
//	    dynconfig.LoadJSON[Config],
//	    nil, nil, nil,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	config := loader.Get()
//	fmt.Printf("DB: %s, Port: %d\n", config.Database, config.Port)
//
// Example with MustLoadAndWatch:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    dynconfig.LoadJSON[*Config],
//	    nil, nil, nil,
//	)
func LoadJSON[T any](file fs.File) (config T, err error) {
	return LoadJSONContext[T](context.Background())(file)
}

// LoadJSONContext returns a load function that works like LoadJSON but reads
// with ctx instead of context.Background().
//
// The returned function uses ctx for every reload, so pass a context that lives
// as long as the Loader: once ctx is canceled, every reload fails.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    dynconfig.LoadJSONContext[Config](ctx),
//	    dynconfig.SaveJSONContext[Config](ctx, "  "),
//	    nil, nil, nil,
//	)
func LoadJSONContext[T any](ctx context.Context) func(file fs.File) (T, error) {
	return func(file fs.File) (config T, err error) {
		err = file.ReadJSON(ctx, &config)
		if err != nil {
			return *new(T), err
		}
		return config, nil
	}
}

// SaveJSON returns a save function that marshals a configuration value of type T
// to JSON and writes it to the file, overwriting any existing content.
//
// The optional indent arguments are concatenated and passed to
// fs.File.WriteJSONIndent: with no arguments (or only empty strings) the JSON
// is written compact, otherwise the concatenated indent is used for line
// indentation.
//
// It is the write counterpart to LoadJSON and is designed to be passed as the
// save function to the constructor for use by Loader.Mutate and Loader.Set.
//
// Type Parameters:
//   - T: The configuration type to marshal to JSON
//
// Example:
//
//	type Config struct {
//	    Database string `json:"database"`
//	    Port     int    `json:"port"`
//	}
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    dynconfig.LoadJSON[Config],
//	    dynconfig.SaveJSON[Config]("  "),
//	    nil, nil, nil,
//	)
//	err := loader.Mutate(false, func(cfg Config) (Config, error) {
//	    cfg.Port = 9090
//	    return cfg, nil
//	})
func SaveJSON[T any](indent ...string) func(file fs.File, config T) error {
	return SaveJSONContext[T](context.Background(), indent...)
}

// SaveJSONContext is like SaveJSON but writes with ctx instead of
// context.Background().
//
// The returned save function uses ctx for every write, so pass a context that
// lives as long as the Loader: once ctx is canceled, every Loader.Mutate and
// Loader.Set through it fails.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    dynconfig.LoadJSONContext[Config](ctx),
//	    dynconfig.SaveJSONContext[Config](ctx, "  "),
//	    nil, nil, nil,
//	)
func SaveJSONContext[T any](ctx context.Context, indent ...string) func(file fs.File, config T) error {
	return func(file fs.File, config T) error {
		return file.WriteJSONIndent(ctx, config, strings.Join(indent, ""))
	}
}
