package dynconfig

import (
	"context"

	"github.com/ungerik/go-fs"
)

// LoadXML returns a function that loads XML configuration from a file.
//
// This is a loader function compatible with LoadAndWatch and MustLoadAndWatch.
// The returned function unmarshals the XML file into a configuration struct of type T.
//
// Type Parameters:
//   - T: The configuration type to unmarshal from XML
//
// Example:
//
//	type Config struct {
//	    XMLName  xml.Name `xml:"config"`
//	    Database string   `xml:"database"`
//	    Port     int      `xml:"port"`
//	}
//
//	loader, err := dynconfig.LoadAndWatch(
//	    "config.xml",
//	    dynconfig.LoadXML[Config],
//	    nil, nil, nil,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	config := loader.Get()
//	fmt.Printf("DB: %s, Port: %d\n", config.Database, config.Port)
func LoadXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}

// LoadEnvXML returns a function that loads XML configuration with environment variable overrides.
//
// This is a loader function compatible with LoadAndWatch and MustLoadAndWatch.
// The returned function:
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
//	    Debug    bool     `xml:"debug"    env:"DEBUG"`
//	}
//
//	// config.xml contains: <config><database>dev.db</database><port>3000</port></config>
//	// Environment has: DB_NAME=prod.db, APP_PORT=8080
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.xml",
//	    dynconfig.LoadEnvXML[*Config],
//	    nil, nil, nil,
//	)
//
//	config := loader.Get()
//	// Result: {Database: "prod.db", Port: 8080, Debug: false}
//	// DB_NAME and APP_PORT from environment override XML values
func LoadEnvXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	err = ParseEnv(&config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}
