package dynconfig

import (
	"strings"
	"unsafe"

	"github.com/ungerik/go-fs"
)

// SplitLines splits a string at newlines.
var SplitLines = func(str string) []string {
	return strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
}

// LoadString reads the passed file as a string.
func LoadString(file fs.File) (string, error) {
	return LoadStringT[string](file)
}

// LoadStringT reads the passed file as a string type T.
func LoadStringT[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(str), nil
}

// LoadStringTrimSpace reads the passed file as a string
// and trims leading and trailing whitespace.
func LoadStringTrimSpace(file fs.File) (string, error) {
	return LoadStringTrimSpaceT[string](file)
}

// LoadStringTrimSpaceT reads the passed file as a string type T
// and trims leading and trailing whitespace.
func LoadStringTrimSpaceT[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(strings.TrimSpace(str)), nil
}

// LoadStringLines parses the passed file as a slice of strings
// by splitting the file content at newlines.
func LoadStringLines(file fs.File) ([]string, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	return SplitLines(str), nil
}

// LoadStringLinesT parses the passed file as a slice of strings of type T
// by splitting the file content at newlines.
func LoadStringLinesT[T ~string](file fs.File) ([]T, error) {
	strs, err := LoadStringLines(file)
	if err != nil {
		return nil, err
	}
	return *(*[]T)(unsafe.Pointer(&strs)), nil //#nosec G103 -- unsafe OK
}

// LoadStringLinesTrimSpace parses the passed file as a slice of strings
// by splitting the file content at newlines
// and trims leading and trailing whitespace from each line.
func LoadStringLinesTrimSpace(file fs.File) ([]string, error) {
	return LoadStringLinesTrimSpaceT[string](file)
}

// LoadStringLinesTrimSpaceT parses the passed file as a slice of strings of type T
// by splitting the file content at newlines
// and trims leading and trailing whitespace from each line.
func LoadStringLinesTrimSpaceT[T ~string](file fs.File) ([]T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := SplitLines(str)
	slice := make([]T, 0, len(strs))
	for _, s := range strs {
		if s = strings.TrimSpace(s); s != "" {
			slice = append(slice, T(s))
		}
	}
	return slice, nil
}

// LoadStringLineSet parses the passed file as a unique set of strings
// by splitting the file content at newlines.
func LoadStringLineSet(file fs.File) (map[string]struct{}, error) {
	return LoadStringLineSetT[string](file)
}

// LoadStringLineSetT parses the passed file as a unique set of strings of type T
// by splitting the file content at newlines.
func LoadStringLineSetT[T ~string](file fs.File) (map[T]struct{}, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := SplitLines(str)
	set := make(map[T]struct{}, len(strs))
	for _, s := range strs {
		set[T(s)] = struct{}{}
	}
	return set, nil
}

// LoadStringLineSetTrimSpace parses the passed file as a unique set of strings
// by splitting the file content at newlines
// and trims leading and trailing whitespace from each line.
// Strings that are empty after trimming whitespace are ignored.
func LoadStringLineSetTrimSpace(file fs.File) (map[string]struct{}, error) {
	return LoadStringLineSetTrimSpaceT[string](file)
}

// LoadStringLineSetTrimSpaceT parses the passed file as a unique set of strings of type T
// by splitting the file content at newlines
// and trims leading and trailing whitespace from each line.
// Strings that are empty after trimming whitespace are ignored.
func LoadStringLineSetTrimSpaceT[T ~string](file fs.File) (map[T]struct{}, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	strs := SplitLines(str)
	set := make(map[T]struct{}, len(strs))
	for _, s := range strs {
		if s = strings.TrimSpace(s); s != "" {
			set[T(s)] = struct{}{}
		}
	}
	return set, nil
}
