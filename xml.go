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

// SaveXML returns a save function that marshals a configuration value of type T
// to XML and writes it to the file, overwriting any existing content.
//
// The optional indent arguments are passed to fs.File.WriteXML: with no
// arguments the XML is written compact, otherwise the concatenated indent
// strings are used for line indentation.
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
	return func(file fs.File, config T) error {
		return file.WriteXML(context.Background(), config, indent...)
	}
}
