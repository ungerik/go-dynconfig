package dynconfig

import (
	"context"

	"github.com/ungerik/go-fs"
)

// LoadJSON unmarshals the passed JSON file into a config of type T.
func LoadJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}

// LoadEnvJSON unmarshals the passed JSON file into a config of type T
// and then parses environment variables into the config
// by looking for struct fields with an `env` tag.
func LoadEnvJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(context.Background(), &config)
	if err != nil {
		return *new(T), err
	}
	err = ParseEnv(&config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}
