package dynconfig

import (
	"context"

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
	err = file.ReadJSON(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}

// SaveJSON returns a save function that marshals a configuration value of type T
// to JSON and writes it to the file, overwriting any existing content.
//
// The optional indent arguments are passed to fs.File.WriteJSON: with no
// arguments the JSON is written compact, otherwise the concatenated indent
// strings are used for line indentation.
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
	return func(file fs.File, config T) error {
		return file.WriteJSON(context.Background(), config, indent...)
	}
}
