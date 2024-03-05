package dynconfig

import (
	"context"

	env "github.com/caarlos0/env/v7"
	"github.com/ungerik/go-fs"
)

// LoadXML unmarshals the passed XML file into a config of type T.
func LoadXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}

// LoadXML unmarshals the passed XML file into a config of type T
// and then parses environment variables into the config
// by looking for struct fields with an `env` tag.
func LoadEnvXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	err = env.Parse(&config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}
