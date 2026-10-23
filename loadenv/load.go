package loadenv

import (
	"context"

	"github.com/ungerik/go-fs"
)

// LoadEnvJSON loads JSON configuration with environment variable overrides.
//
// This is a loader function compatible with dynconfig.LoadAndWatch and
// MustLoadAndWatch. It:
//  1. Unmarshals the JSON file into a configuration struct of type T
//  2. Overrides values with environment variables based on `env` struct tags
//
// Environment variables are parsed using the `env` struct tag.
// See ParseEnv for details on struct tag format and supported types.
//
// Type Parameters:
//   - T: The configuration type to unmarshal from JSON
//
// Example:
//
//	type Config struct {
//	    Database string `json:"database" env:"DB_NAME"`
//	    Port     int    `json:"port"     env:"APP_PORT"`
//	    Debug    bool   `json:"debug"    env:"DEBUG"`
//	}
//
//	// config.json contains: {"database": "dev.db", "port": 3000, "debug": false}
//	// Environment has: DB_NAME=prod.db, APP_PORT=8080
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    loadenv.LoadEnvJSON[*Config],
//	    nil, nil, nil, nil,
//	)
//
//	config := loader.Get()
//	// Result: {Database: "prod.db", Port: 8080, Debug: false}
//	// DB_NAME and APP_PORT from environment override JSON values
//
// Example with error handling:
//
//	loader, err := dynconfig.LoadAndWatch(
//	    "config.json",
//	    loadenv.LoadEnvJSON[Config],
//	    nil,
//	    func(err error) Config {
//	        log.Printf("Config error: %v", err)
//	        return Config{Database: "fallback.db", Port: 8080}
//	    },
//	    nil,
//	)
func LoadEnvJSON[T any](file fs.File) (config T, err error) {
	return LoadEnvJSONContext[T](context.Background())(file)
}

// LoadEnvJSONContext returns a load function that works like LoadEnvJSON but
// reads with ctx instead of context.Background().
//
// The returned function uses ctx for every reload, so pass a context that lives
// as long as the Loader: once ctx is canceled, every reload fails.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.json",
//	    loadenv.LoadEnvJSONContext[Config](ctx),
//	    nil, nil, nil, nil,
//	)
func LoadEnvJSONContext[T any](ctx context.Context) func(file fs.File) (T, error) {
	return func(file fs.File) (config T, err error) {
		err = file.ReadJSON(ctx, &config)
		if err != nil {
			return *new(T), err
		}
		err = ParseEnv(&config)
		if err != nil {
			return *new(T), err
		}
		return config, nil
	}
}

// LoadEnvXML loads XML configuration with environment variable overrides.
//
// This is a loader function compatible with dynconfig.LoadAndWatch and
// MustLoadAndWatch. It:
//  1. Unmarshals the XML file into a configuration struct of type T
//  2. Overrides values with environment variables based on `env` struct tags
//
// Environment variables are parsed using the `env` struct tag.
// See ParseEnv for details on struct tag format and supported types.
//
// Type Parameters:
//   - T: The configuration type to unmarshal from XML
//
// Example:
//
//	type Config struct {
//	    XMLName  xml.Name `xml:"config"`
//	    Database string   `xml:"database" env:"DB_NAME"`
//	    Port     int      `xml:"port"     env:"APP_PORT"`
//	    Debug    bool     `xml:"debug"     env:"DEBUG"`
//	}
//
//	// config.xml contains: <config><database>dev.db</database><port>3000</port></config>
//	// Environment has: DB_NAME=prod.db, APP_PORT=8080
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.xml",
//	    loadenv.LoadEnvXML[*Config],
//	    nil, nil, nil, nil,
//	)
//
//	config := loader.Get()
//	// Result: {Database: "prod.db", Port: 8080, Debug: false}
//	// DB_NAME and APP_PORT from environment override XML values
func LoadEnvXML[T any](file fs.File) (config T, err error) {
	return LoadEnvXMLContext[T](context.Background())(file)
}

// LoadEnvXMLContext returns a load function that works like LoadEnvXML but
// reads with ctx instead of context.Background(). See LoadEnvJSONContext for
// the lifetime of ctx.
func LoadEnvXMLContext[T any](ctx context.Context) func(file fs.File) (T, error) {
	return func(file fs.File) (config T, err error) {
		err = file.ReadXML(ctx, &config)
		if err != nil {
			return *new(T), err
		}
		err = ParseEnv(&config)
		if err != nil {
			return *new(T), err
		}
		return config, nil
	}
}
