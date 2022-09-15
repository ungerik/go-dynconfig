package dynconfig

import "github.com/ungerik/go-fs"

func LoadXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(&config)
	return config, err
}
