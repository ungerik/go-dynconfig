package dynconfig

import (
	"strings"
	"unsafe"

	"github.com/ungerik/go-fs"
)

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
