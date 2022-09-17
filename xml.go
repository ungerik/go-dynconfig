package dynconfig

import (
	"context"

	"github.com/ungerik/go-fs"
)

func LoadXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	return config, err
}
