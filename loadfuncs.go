package dynconfig

import (
	"context"
	"strings"
	"unsafe"

	"github.com/ungerik/go-fs"
)

func LoadJSON[T any](file fs.File) (config T, err error) {
	err = file.ReadJSON(context.Background(), &config)
	return config, err
}

func LoadXML[T any](file fs.File) (config T, err error) {
	err = file.ReadXML(context.Background(), &config)
	return config, err
}

func LoadString[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return "", err
	}
	return T(str), nil
}

func LoadStringTrimSpace[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return "", err
	}
	return T(strings.TrimSpace(str)), nil
}

func LoadStringLines[T []S, S ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	return *(*T)(unsafe.Pointer(&strs)), nil //#nosec G103 -- unsafe OK
}

func LoadStringLinesTrimSpace[T []S, S ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString(context.Background())
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
	str, err := file.ReadAllString(context.Background())
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
	str, err := file.ReadAllString(context.Background())
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
