package dynconfig

import (
	"encoding"
	"fmt"
	"strings"

	"github.com/ungerik/go-fs"
)

func LoadTXT[T any](file fs.File) (config T, err error) {
	var data []byte
	data, err = file.ReadAll()
	if err != nil {
		return config, err
	}
	switch configPtr := any(&config).(type) {
	case *string:
		*configPtr = string(data)

	case *[]string:
		*configPtr = splitLines(string(data))

	case *map[string]struct{}:
		strs := splitLines(string(data))
		m := make(map[string]struct{}, len(strs))
		for _, s := range strs {
			m[s] = struct{}{}
		}
		*configPtr = m

	case *[]byte:
		*configPtr = data

	case encoding.TextUnmarshaler:
		err = configPtr.UnmarshalText(data)
		if err != nil {
			return config, err
		}

	default:
		return config, fmt.Errorf("type %T not supported by TXTLoader", config)
	}
	return config, nil
}

func splitLines(str string) []string {
	return strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
}
