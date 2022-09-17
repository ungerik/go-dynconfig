package dynconfig

import (
	"context"

	"github.com/ungerik/go-fs"
)

func LoadJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(context.Background(), &config)
	return config, err
}
