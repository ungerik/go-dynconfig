package dynconfig

import (
	"context"
	"strings"
	"unsafe"

	env "github.com/caarlos0/env/v7"
	fs "github.com/ungerik/go-fs"
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
	err = env.Parse(&config)
	if err != nil {
		return *new(T), err
	}
	return config, nil
}

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

// LoadString reads the passed file as a string type T.
func LoadString[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(str), nil
}

// LoadStringTrimSpace reads the passed file as a string type T
// and trims leading and trailing whitespace.
func LoadStringTrimSpace[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(strings.TrimSpace(str)), nil
}

func LoadStringLines[T []S, S ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	return *(*T)(unsafe.Pointer(&strs)), nil //#nosec G103 -- unsafe OK
}

func LoadStringLinesTrimSpace[T []S, S ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	slice := make(T, 0, len(strs))
	for _, s := range strs {
		if s = strings.TrimSpace(s); s != "" {
			slice = append(slice, S(s))
		}
	}
	return slice, nil
}

func LoadStringLineSet[T ~map[S]struct{}, S ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	set := make(T, len(strs))
	for _, s := range strs {
		set[S(s)] = struct{}{}
	}
	return set, nil
}

func LoadStringLineSetTrimSpace[T ~map[S]struct{}, S ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	set := make(T, len(strs))
	for _, s := range strs {
		if s = strings.TrimSpace(s); s != "" {
			set[S(s)] = struct{}{}
		}
	}
	return set, nil
}

func splitLines(str string) []string {
	return strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
}
