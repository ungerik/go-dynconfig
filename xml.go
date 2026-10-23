package dynconfig

import (
	"context"
	"strings"

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
	return LoadXMLContext[T](context.Background())(file)
}

// LoadXMLContext returns a load function that works like LoadXML but reads with
// ctx instead of context.Background(). See LoadJSONContext for the lifetime of
// ctx.
func LoadXMLContext[T any](ctx context.Context) func(file fs.File) (T, error) {
	return func(file fs.File) (config T, err error) {
		err = file.ReadXML(ctx, &config)
		if err != nil {
			return *new(T), err
		}
		return config, nil
	}
}

// SaveXML returns a save function that marshals a configuration value of type T
// to XML and writes it to the file, overwriting any existing content.
//
// The optional indent arguments are concatenated and passed to
// fs.File.WriteXMLIndent: with no arguments (or only empty strings) the XML
// is written compact, otherwise the concatenated indent is used for line
// indentation.
//
// It is the write counterpart to LoadXML and is designed to be passed as the
// save function to the constructor for use by Loader.Mutate and Loader.Set.
//
// Type Parameters:
//   - T: The configuration type to marshal to XML
//
// Example:
//
//	type Config struct {
//	    XMLName  xml.Name `xml:"config"`
//	    Database string   `xml:"database"`
//	    Port     int      `xml:"port"`
//	}
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.xml",
//	    dynconfig.LoadXML[Config],
//	    dynconfig.SaveXML[Config]("  "),
//	    nil, nil, nil,
//	)
//	err := loader.Mutate(false, func(cfg Config) (Config, error) {
//	    cfg.Port = 9090
//	    return cfg, nil
//	})
func SaveXML[T any](indent ...string) func(file fs.File, config T) error {
	return SaveXMLContext[T](context.Background(), indent...)
}

// SaveXMLContext is like SaveXML but writes with ctx instead of
// context.Background().
//
// The returned save function uses ctx for every write, so pass a context that
// lives as long as the Loader: once ctx is canceled, every Loader.Mutate and
// Loader.Set through it fails.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "config.xml",
//	    dynconfig.LoadXMLContext[Config](ctx),
//	    dynconfig.SaveXMLContext[Config](ctx, "  "),
//	    nil, nil, nil,
//	)
func SaveXMLContext[T any](ctx context.Context, indent ...string) func(file fs.File, config T) error {
	return func(file fs.File, config T) error {
		return file.WriteXMLIndent(ctx, config, strings.Join(indent, ""))
	}
}
