package dynconfig

import (
	"context"
	"strings"
	"unsafe"

	"github.com/ungerik/go-fs"
)

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

func LoadStringLines[T ~string](file fs.File) ([]T, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	return *(*[]T)(unsafe.Pointer(&strs)), nil
}

func LoadStringLinesTrimSpace[T ~string](file fs.File) ([]T, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	result := make([]T, 0, len(strs))
	for _, s := range strs {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, T(s))
		}
	}
	return result, nil
}

func LoadStringLineSet[T ~string](file fs.File) (map[T]struct{}, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	set := make(map[T]struct{}, len(strs))
	for _, s := range strs {
		set[T(s)] = struct{}{}
	}
	return set, nil
}

func LoadStringLineSetTrimSpace[T ~string](file fs.File) (map[T]struct{}, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return nil, err
	}
	strs := splitLines(str)
	set := make(map[T]struct{}, len(strs))
	for _, s := range strs {
		set[T(strings.TrimSpace(s))] = struct{}{}
	}
	return set, nil
}

func splitLines(str string) []string {
	return strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
}
