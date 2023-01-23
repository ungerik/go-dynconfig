package dynconfig

import (
	"context"
	"strings"
	"unsafe"

	env "github.com/caarlos0/env/v6"
	fs "github.com/ungerik/go-fs"
)

func LoadJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(context.Background(), &config)
	if err != nil {
		var zero T
		return zero, err
	}
	return config, nil
}

func LoadEnvJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(context.Background(), &config)
	if err != nil {
		var zero T
		return zero, err
	}
	err = env.Parse(&config)
	if err != nil {
		var zero T
		return zero, err
	}
	return config, nil
}

func LoadXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	if err != nil {
		var zero T
		return zero, err
	}
	return config, nil
}

func LoadEnvXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	if err != nil {
		var zero T
		return zero, err
	}
	err = env.Parse(&config)
	if err != nil {
		var zero T
		return zero, err
	}
	return config, nil
}

func LoadString[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(str), nil
}

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
