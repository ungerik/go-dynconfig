package dynconfig

import (
	"strings"
	"unsafe"

	"github.com/ungerik/go-fs"
)

// SplitLines is a configurable function that splits a string at newlines.
//
// The default implementation splits on both '\n' and '\r' characters,
// effectively handling Unix (\n), Windows (\r\n), and old Mac (\r) line endings.
//
// You can replace this function with a custom implementation for different
// line splitting behavior.
//
// Example custom implementation:
//
//	// Only split on Unix newlines
//	dynconfig.SplitLines = func(str string) []string {
//	    return strings.Split(str, "\n")
//	}
//
// Used by:
//   - LoadStringLines and LoadStringLinesT
//   - LoadStringLinesTrimSpace and LoadStringLinesTrimSpaceT
//   - LoadStringLineSet and LoadStringLineSetT
//   - LoadStringLineSetTrimSpace and LoadStringLineSetTrimSpaceT
var SplitLines = func(str string) []string {
	return strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
}

// LoadString loads the entire file content as a string.
//
// This is the base text loading function. For more specialized loading,
// see LoadStringTrimSpace, LoadStringLines, or LoadStringLineSet.
//
// Example:
//
//	content, err := dynconfig.LoadString("data.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(content)
func LoadString(file fs.File) (string, error) {
	return LoadStringT[string](file)
}

// LoadStringT loads the entire file content as a string of custom type T.
//
// Type T must be a string type (e.g., type MyString string).
// This is useful for type-safe string configurations.
//
// Example:
//
//	type APIKey string
//
//	key, err := dynconfig.LoadStringT[APIKey]("api-key.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// key is of type APIKey
func LoadStringT[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(str), nil
}

// LoadStringTrimSpace loads the file content as a string with whitespace trimmed.
//
// Removes leading and trailing whitespace (spaces, tabs, newlines) from the content.
//
// Example:
//
//	// token.txt contains: "  abc123  \n"
//	token, err := dynconfig.LoadStringTrimSpace("token.txt")
//	// token will be "abc123"
func LoadStringTrimSpace(file fs.File) (string, error) {
	return LoadStringTrimSpaceT[string](file)
}

// LoadStringTrimSpaceT loads the file content as a string type T with whitespace trimmed.
//
// Combines LoadStringT with whitespace trimming.
//
// Example:
//
//	type Token string
//
//	// auth-token.txt contains: "  secret123  \n"
//	token, err := dynconfig.LoadStringTrimSpaceT[Token]("auth-token.txt")
//	// token will be Token("secret123")
func LoadStringTrimSpaceT[T ~string](file fs.File) (T, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return "", err
	}
	return T(strings.TrimSpace(str)), nil
}

// LoadStringLines loads the file as a slice of strings, one per line.
//
// Lines are split using the SplitLines function, which handles Unix, Windows,
// and old Mac line endings by default.
//
// Example:
//
//	// hosts.txt contains:
//	// api.example.com
//	// db.example.com
//	// cache.example.com
//
//	hosts, err := dynconfig.LoadStringLines("hosts.txt")
//	// hosts = []string{"api.example.com", "db.example.com", "cache.example.com"}
func LoadStringLines(file fs.File) ([]string, error) {
	str, err := file.ReadAllString()
	if err != nil {
		return nil, err
	}
	return SplitLines(str), nil
}

// LoadStringLinesT loads the file as a slice of strings of type T, one per line.
//
// Type T must be a string type. Useful for type-safe lists.
//
// Example:
//
//	type Hostname string
//
//	hosts, err := dynconfig.LoadStringLinesT[Hostname]("servers.txt")
//	// Returns []Hostname
func LoadStringLinesT[T ~string](file fs.File) ([]T, error) {
	strs, err := LoadStringLines(file)
	if err != nil {
		return nil, err
	}
	return *(*[]T)(unsafe.Pointer(&strs)), nil //#nosec G103 -- unsafe OK
}

// LoadStringLinesTrimSpace loads the file as a slice of strings with trimmed whitespace.
//
// Each line has leading and trailing whitespace removed.
// Empty lines (after trimming) are excluded from the result.
//
// Example:
//
//	// config.txt contains:
//	//   api.example.com
//	//
//	//   db.example.com
//
//	lines, err := dynconfig.LoadStringLinesTrimSpace("config.txt")
//	// lines = []string{"api.example.com", "db.example.com"}
func LoadStringLinesTrimSpace(file fs.File) ([]string, error) {
	return LoadStringLinesTrimSpaceT[string](file)
}

// LoadStringLinesTrimSpaceT loads the file as a slice of type T with trimmed whitespace.
//
// Combines line splitting with whitespace trimming and empty line removal.
//
// Example:
//
//	type Email string
//
//	// emails.txt contains:
//	//   user@example.com
//	//
//	//   admin@example.com
//
//	emails, err := dynconfig.LoadStringLinesTrimSpaceT[Email]("emails.txt")
//	// emails = []Email{"user@example.com", "admin@example.com"}
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

// LoadStringLineSet loads the file as a unique set of strings, one per line.
//
// Returns a map where each line is a key with an empty struct value.
// This is useful for membership checking (e.g., blacklists, whitelists).
// Duplicate lines result in a single entry.
//
// Example:
//
//	// blocked.txt contains:
//	// spam.com
//	// spam.com
//	// malicious.org
//
//	blocked, err := dynconfig.LoadStringLineSet("blocked.txt")
//	// blocked = map[string]struct{}{"spam.com": {}, "malicious.org": {}}
//
//	// Check membership
//	if _, isBlocked := blocked["spam.com"]; isBlocked {
//	    // Domain is blocked
//	}
func LoadStringLineSet(file fs.File) (map[string]struct{}, error) {
	return LoadStringLineSetT[string](file)
}

// LoadStringLineSetT loads the file as a unique set of strings of type T.
//
// Type T must be a string type. Useful for type-safe sets.
//
// Example:
//
//	type Domain string
//
//	allowed, err := dynconfig.LoadStringLineSetT[Domain]("allowed-domains.txt")
//	// Returns map[Domain]struct{}
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

// LoadStringLineSetTrimSpace loads the file as a unique set with trimmed whitespace.
//
// Each line has whitespace removed before being added to the set.
// Empty lines (after trimming) are ignored.
// Duplicate lines (after trimming) result in a single entry.
//
// Example:
//
//	// whitelist.txt contains:
//	//   trusted.com
//	//
//	//   safe.org
//	//   trusted.com
//
//	whitelist, err := dynconfig.LoadStringLineSetTrimSpace("whitelist.txt")
//	// whitelist = map[string]struct{}{"trusted.com": {}, "safe.org": {}}
func LoadStringLineSetTrimSpace(file fs.File) (map[string]struct{}, error) {
	return LoadStringLineSetTrimSpaceT[string](file)
}

// LoadStringLineSetTrimSpaceT loads the file as a unique set of type T with trimmed whitespace.
//
// Combines set creation with whitespace trimming and empty line removal.
//
// Example:
//
//	type IPAddress string
//
//	// blocked-ips.txt contains:
//	//   192.168.1.1
//	//   10.0.0.1
//	//   192.168.1.1
//
//	blocked, err := dynconfig.LoadStringLineSetTrimSpaceT[IPAddress]("blocked-ips.txt")
//	// blocked = map[IPAddress]struct{}{"192.168.1.1": {}, "10.0.0.1": {}}
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
