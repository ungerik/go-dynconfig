package dynconfig

import "github.com/ungerik/go-fs"

func LoadJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(&config)
	return config, err
}
