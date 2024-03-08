package dynconfig

import (
	"reflect"

	env "github.com/caarlos0/env/v7"
)

// ParseEnv parses environment variables into the passed struct
// pointer by looking for struct fields with an `env` tag.
//
// This global configuration function is used by
// LoadEnvJSON and LoadEnvXML.
var ParseEnv = func(dest any) error {
	// Deref pointer to pointer because env.Parse
	// only accepts pointers to structs
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Ptr {
		dest = v.Elem().Interface()
	}
	return env.Parse(dest)
}
