package dynconfig

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unsafe"

	"github.com/ungerik/go-fs"
)

// utf8BOM is the UTF-8 encoded byte order mark U+FEFF that some editors write
// at the start of a text file.
const utf8BOM = "\uFEFF"

// readAllStringWithoutBOM reads the whole file as a string and removes a UTF-8
// byte order mark at its start. All text loaders read files with it.
//
// Only UTF-8 is handled: the BOMs of other encodings like UTF-16 or UTF-32 are
// not detected, and such files are returned as raw bytes without decoding.
func readAllStringWithoutBOM(file fs.File) (string, error) {
	str, err := file.ReadAllString(context.Background())
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(str, utf8BOM), nil
}

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
//   - LoadStringLinesFormat and LoadStringLinesFormatT
//   - LoadStringLineSetFormat and LoadStringLineSetFormatT
//
// These loaders remove a UTF-8 byte order mark at the start of the file before
// they call SplitLines, so the BOM never becomes part of the first line.
var SplitLines = func(str string) []string {
	// FieldsFunc splits at each run of matching runes and never emits empty
	// fields, so a "\r\n" pair counts as a single line break (not two) and
	// blank lines and leading/trailing line breaks are dropped.
	return strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
}

// LoadString loads the entire file content as a string.
//
// The file is read as UTF-8. A UTF-8 byte order mark at the start of the file
// is removed, like all text loaders of this package do, and SaveString does not
// write one back. Other encodings are not supported: a UTF-16 or UTF-32 file is
// not detected by its byte order mark or decoded, but returned as raw bytes.
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

// SaveString writes the entire string as the file content, overwriting any
// existing content.
//
// It is the write counterpart to LoadString and can be passed directly as the
// save function to the constructor (NewLoader, LoadAndWatch, MustLoadAndWatch)
// for use by Loader.Mutate and Loader.Set. No configuration is needed, so it is a
// save function itself rather than a factory returning one.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "message.txt",
//	    dynconfig.LoadString,
//	    dynconfig.SaveString,
//	    nil, nil, nil,
//	)
//	err := loader.Mutate(false, func(s string) (string, error) {
//	    return s + "!", nil
//	})
func SaveString(file fs.File, config string) error {
	return file.WriteAllString(context.Background(), config)
}

// LoadStringT loads the entire file content as a string of custom type T.
//
// Type T must be a string type (e.g., type MyString string).
// This is useful for type-safe string configurations.
// Like LoadString, it removes a UTF-8 byte order mark at the start of the file.
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
	str, err := readAllStringWithoutBOM(file)
	if err != nil {
		return "", err
	}
	return T(str), nil
}

// SaveStringT writes the entire string of custom type T as the file content,
// overwriting any existing content.
//
// Type T must be a string type (e.g., type MyString string).
// It is the write counterpart to LoadStringT and delegates to SaveString.
//
// Example:
//
//	type APIKey string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "api-key.txt",
//	    dynconfig.LoadStringT[APIKey],
//	    dynconfig.SaveStringT[APIKey],
//	    nil, nil, nil,
//	)
func SaveStringT[T ~string](file fs.File, config T) error {
	return SaveString(file, string(config))
}

// LoadStringTrimSpace loads the file content as a string with whitespace trimmed.
//
// Removes leading and trailing whitespace (spaces, tabs, newlines) from the
// content, and a UTF-8 byte order mark at the start of the file, which
// strings.TrimSpace does not count as whitespace.
//
// Example:
//
//	// token.txt contains: "  abc123  \n"
//	token, err := dynconfig.LoadStringTrimSpace("token.txt")
//	// token will be "abc123"
func LoadStringTrimSpace(file fs.File) (string, error) {
	return LoadStringTrimSpaceT[string](file)
}

// SaveStringTrimSpace writes the string as the file content with leading and
// trailing whitespace removed, overwriting any existing content.
//
// It is the write counterpart to LoadStringTrimSpace: the value is trimmed on
// save so the on-disk content matches what the loader would read back.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "token.txt",
//	    dynconfig.LoadStringTrimSpace,
//	    dynconfig.SaveStringTrimSpace,
//	    nil, nil, nil,
//	)
func SaveStringTrimSpace(file fs.File, config string) error {
	return file.WriteAllString(context.Background(), strings.TrimSpace(config))
}

// LoadStringTrimSpaceT loads the file content as a string type T with whitespace trimmed.
//
// Combines LoadStringT with whitespace trimming. Like LoadStringTrimSpace, it
// also removes a UTF-8 byte order mark at the start of the file.
//
// Example:
//
//	type Token string
//
//	// auth-token.txt contains: "  secret123  \n"
//	token, err := dynconfig.LoadStringTrimSpaceT[Token]("auth-token.txt")
//	// token will be Token("secret123")
func LoadStringTrimSpaceT[T ~string](file fs.File) (T, error) {
	str, err := readAllStringWithoutBOM(file)
	if err != nil {
		return "", err
	}
	return T(strings.TrimSpace(str)), nil
}

// SaveStringTrimSpaceT writes the string of type T as the file content with
// leading and trailing whitespace removed, overwriting any existing content.
//
// Type T must be a string type.
// It is the write counterpart to LoadStringTrimSpaceT and delegates to
// SaveStringTrimSpace.
//
// Example:
//
//	type Token string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "auth-token.txt",
//	    dynconfig.LoadStringTrimSpaceT[Token],
//	    dynconfig.SaveStringTrimSpaceT[Token],
//	    nil, nil, nil,
//	)
func SaveStringTrimSpaceT[T ~string](file fs.File, config T) error {
	return SaveStringTrimSpace(file, string(config))
}

// LoadStringLines loads the file as a slice of strings, one per line.
//
// Lines are split using the SplitLines function, which handles Unix, Windows,
// and old Mac line endings by default. A UTF-8 byte order mark at the start of
// the file is removed first.
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
	return loadLines[string](file, lineFilter{})
}

// SaveStringLines returns a save function that writes a slice of strings to the
// file, one per line, overwriting any existing content.
//
// The optional sep arguments define the separator written between lines; with
// no arguments lines are separated by a single newline ("\n"), otherwise the
// concatenated sep strings are used. The default newline separator round-trips
// with LoadStringLines, which splits on newlines via SplitLines.
//
// To keep that round-trip intact, any occurrence of the separator inside a line
// value is replaced with a space before writing (or removed when the separator
// is itself a space), so a value can never be split across multiple lines when
// the file is read back.
//
// It is the write counterpart to LoadStringLines and is designed to be passed
// as the save function to the constructor for use by Loader.Mutate and
// Loader.Set. Because it needs the separator as configuration, it is a factory
// returning the save function rather than a save function itself.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "hosts.txt",
//	    dynconfig.LoadStringLines,
//	    dynconfig.SaveStringLines(),
//	    nil, nil, nil,
//	)
//	err := loader.Mutate(false, func(hosts []string) ([]string, error) {
//	    return append(hosts, "new.example.com"), nil
//	})
func SaveStringLines(sep ...string) func(file fs.File, config []string) error {
	separator := resolveLineSeparator(sep)
	return func(file fs.File, config []string) error {
		lines := make([]string, len(config))
		for i, line := range config {
			lines[i] = sanitizeLine(line, separator)
		}
		return file.WriteAllString(context.Background(), strings.Join(lines, separator))
	}
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
	return loadLines[T](file, lineFilter{})
}

// SaveStringLinesT returns a save function that writes a slice of strings of
// type T to the file, one per line, overwriting any existing content.
//
// Type T must be a string type. See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLinesT and delegates to
// SaveStringLines.
//
// Example:
//
//	type Hostname string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "servers.txt",
//	    dynconfig.LoadStringLinesT[Hostname],
//	    dynconfig.SaveStringLinesT[Hostname](),
//	    nil, nil, nil,
//	)
func SaveStringLinesT[T ~string](sep ...string) func(file fs.File, config []T) error {
	save := SaveStringLines(sep...)
	return func(file fs.File, config []T) error {
		return save(file, *(*[]string)(unsafe.Pointer(&config))) //#nosec G103 -- unsafe OK
	}
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

// SaveStringLinesTrimSpace returns a save function that writes a slice of
// strings to the file, one per line, with each line trimmed of leading and
// trailing whitespace and empty lines dropped, overwriting any existing content.
//
// See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLinesTrimSpace.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "emails.txt",
//	    dynconfig.LoadStringLinesTrimSpace,
//	    dynconfig.SaveStringLinesTrimSpace(),
//	    nil, nil, nil,
//	)
func SaveStringLinesTrimSpace(sep ...string) func(file fs.File, config []string) error {
	separator := resolveLineSeparator(sep)
	return func(file fs.File, config []string) error {
		lines := make([]string, 0, len(config))
		for _, line := range config {
			// Sanitize first so spaces introduced by replacing a separator
			// character at a value's boundary are trimmed away too.
			if s := strings.TrimSpace(sanitizeLine(line, separator)); s != "" {
				lines = append(lines, s)
			}
		}
		return file.WriteAllString(context.Background(), strings.Join(lines, separator))
	}
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
	return loadLines[T](file, lineFilter{trimSpace: true})
}

// SaveStringLinesTrimSpaceT returns a save function that writes a slice of type
// T to the file, one per line, with each line trimmed of leading and trailing
// whitespace and empty lines dropped, overwriting any existing content.
//
// Type T must be a string type. See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLinesTrimSpaceT and delegates to
// SaveStringLinesTrimSpace.
//
// Example:
//
//	type Email string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "emails.txt",
//	    dynconfig.LoadStringLinesTrimSpaceT[Email],
//	    dynconfig.SaveStringLinesTrimSpaceT[Email](),
//	    nil, nil, nil,
//	)
func SaveStringLinesTrimSpaceT[T ~string](sep ...string) func(file fs.File, config []T) error {
	save := SaveStringLinesTrimSpace(sep...)
	return func(file fs.File, config []T) error {
		return save(file, *(*[]string)(unsafe.Pointer(&config))) //#nosec G103 -- unsafe OK
	}
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

// SaveStringLineSet returns a save function that writes a set of strings to the
// file, one per line, overwriting any existing content. The lines are sorted so
// the output is deterministic regardless of map iteration order.
//
// See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLineSet.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "blocked.txt",
//	    dynconfig.LoadStringLineSet,
//	    dynconfig.SaveStringLineSet(),
//	    nil, nil, nil,
//	)
//	err := loader.Mutate(false, func(set map[string]struct{}) (map[string]struct{}, error) {
//	    set["spam.com"] = struct{}{}
//	    return set, nil
//	})
func SaveStringLineSet(sep ...string) func(file fs.File, config map[string]struct{}) error {
	separator := resolveLineSeparator(sep)
	return func(file fs.File, config map[string]struct{}) error {
		lines := make([]string, 0, len(config))
		for line := range config {
			lines = append(lines, sanitizeLine(line, separator))
		}
		slices.Sort(lines)
		lines = slices.Compact(lines) // Drop duplicates that collide after sanitizing
		return file.WriteAllString(context.Background(), strings.Join(lines, separator))
	}
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
	return loadLineSet[T](file, lineFilter{})
}

// SaveStringLineSetT returns a save function that writes a set of strings of
// type T to the file, one per line, overwriting any existing content. The lines
// are sorted so the output is deterministic regardless of map iteration order.
//
// Type T must be a string type. See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLineSetT and delegates to
// SaveStringLineSet.
//
// Example:
//
//	type Domain string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "allowed-domains.txt",
//	    dynconfig.LoadStringLineSetT[Domain],
//	    dynconfig.SaveStringLineSetT[Domain](),
//	    nil, nil, nil,
//	)
func SaveStringLineSetT[T ~string](sep ...string) func(file fs.File, config map[T]struct{}) error {
	save := SaveStringLineSet(sep...)
	return func(file fs.File, config map[T]struct{}) error {
		set := make(map[string]struct{}, len(config))
		for line := range config {
			set[string(line)] = struct{}{}
		}
		return save(file, set)
	}
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

// SaveStringLineSetTrimSpace returns a save function that writes a set of
// strings to the file, one per line, with each line trimmed of leading and
// trailing whitespace and empty lines dropped, overwriting any existing content.
// The lines are sorted so the output is deterministic regardless of map
// iteration order.
//
// See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLineSetTrimSpace.
//
// Example:
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "whitelist.txt",
//	    dynconfig.LoadStringLineSetTrimSpace,
//	    dynconfig.SaveStringLineSetTrimSpace(),
//	    nil, nil, nil,
//	)
func SaveStringLineSetTrimSpace(sep ...string) func(file fs.File, config map[string]struct{}) error {
	separator := resolveLineSeparator(sep)
	return func(file fs.File, config map[string]struct{}) error {
		lines := make([]string, 0, len(config))
		for line := range config {
			// Sanitize first so spaces introduced by replacing a separator
			// character at a value's boundary are trimmed away too.
			if s := strings.TrimSpace(sanitizeLine(line, separator)); s != "" {
				lines = append(lines, s)
			}
		}
		slices.Sort(lines)
		lines = slices.Compact(lines) // Drop duplicates that collide after sanitizing and trimming
		return file.WriteAllString(context.Background(), strings.Join(lines, separator))
	}
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
	return loadLineSet[T](file, lineFilter{trimSpace: true})
}

// SaveStringLineSetTrimSpaceT returns a save function that writes a set of
// strings of type T to the file, one per line, with each line trimmed of
// leading and trailing whitespace and empty lines dropped, overwriting any
// existing content. The lines are sorted so the output is deterministic
// regardless of map iteration order, and duplicates that collide after trimming
// are written only once.
//
// Type T must be a string type. See SaveStringLines for the separator handling.
// It is the write counterpart to LoadStringLineSetTrimSpaceT and delegates to
// SaveStringLineSetTrimSpace.
//
// Example:
//
//	type IPAddress string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "blocked-ips.txt",
//	    dynconfig.LoadStringLineSetTrimSpaceT[IPAddress],
//	    dynconfig.SaveStringLineSetTrimSpaceT[IPAddress](),
//	    nil, nil, nil,
//	)
func SaveStringLineSetTrimSpaceT[T ~string](sep ...string) func(file fs.File, config map[T]struct{}) error {
	save := SaveStringLineSetTrimSpace(sep...)
	return func(file fs.File, config map[T]struct{}) error {
		set := make(map[string]struct{}, len(config))
		for line := range config {
			set[string(line)] = struct{}{}
		}
		return save(file, set)
	}
}

// LineFormat configures how the line loaders LoadStringLinesFormat,
// LoadStringLinesFormatT, LoadStringLineSetFormat and LoadStringLineSetFormatT
// turn the lines of a file into values.
//
// The zero value reads the lines like LoadStringLines and LoadStringLineSet.
// Like all line loaders, the LineFormat loaders remove a UTF-8 byte order mark
// at the start of the file, so it can not hide a comment on the first line.
type LineFormat struct {
	// TrimSpace trims leading and trailing whitespace from every line and
	// skips lines that are empty after trimming, like LoadStringLinesTrimSpace
	// and LoadStringLineSetTrimSpace.
	TrimSpace bool

	// CommentMarkers are the line comment markers, for example "#" or "//".
	// No markers means no comments.
	//
	// Everything from the first occurrence of any marker to the end of the
	// line is a comment. The comment and the whitespace in front of it are
	// removed before the line is used as a value, and lines that are empty
	// after that are skipped. So a line holding only a comment, even an
	// indented one, yields no value, and "spam.com   # reported" yields
	// "spam.com" also without TrimSpace.
	//
	// There is no escape syntax: a value can not contain a marker. Choose
	// markers that do not occur in values, for example "//" cuts URLs like
	// "https://example.com" after "https:".
	//
	// An empty marker, or one containing a line break ('\n' or '\r'), is an
	// error that the load function returns.
	//
	// Comments are only understood when reading. A save function like
	// SaveStringLines writes the values it is given, so Loader.Mutate and
	// Loader.Set overwrite the file without its comments, and a value
	// containing a marker is cut at the marker when the file is read back.
	CommentMarkers []string
}

// LoadStringLinesFormat returns a load function that reads the file as a slice
// of strings, one per line, handling whitespace and line comments as configured
// by format.
//
// Because it needs the format as configuration, it is a factory returning the
// load function rather than a load function itself.
//
// Example:
//
//	// emails.txt contains:
//	// # admins
//	//   admin@example.com   # on call
//	//
//	//   user@example.com
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "emails.txt",
//	    dynconfig.LoadStringLinesFormat(dynconfig.LineFormat{
//	        TrimSpace:      true,
//	        CommentMarkers: []string{"#"},
//	    }),
//	    dynconfig.SaveStringLinesTrimSpace(), // writes without comments
//	    nil, nil, nil,
//	)
//	// loader.Get() = []string{"admin@example.com", "user@example.com"}
func LoadStringLinesFormat(format LineFormat) func(file fs.File) ([]string, error) {
	return LoadStringLinesFormatT[string](format)
}

// LoadStringLinesFormatT returns a load function that reads the file as a slice
// of strings of type T, one per line, handling whitespace and line comments as
// configured by format.
//
// Type T must be a string type.
//
// Example:
//
//	type Hostname string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "servers.txt",
//	    dynconfig.LoadStringLinesFormatT[Hostname](dynconfig.LineFormat{
//	        CommentMarkers: []string{"#", "//"},
//	    }),
//	    nil, nil, nil, nil,
//	)
func LoadStringLinesFormatT[T ~string](format LineFormat) func(file fs.File) ([]T, error) {
	filter, err := newLineFilter(format)
	return func(file fs.File) ([]T, error) {
		if err != nil {
			return nil, err
		}
		return loadLines[T](file, filter)
	}
}

// LoadStringLineSetFormat returns a load function that reads the file as a
// unique set of strings, one per line, handling whitespace and line comments as
// configured by format.
//
// Lines that are equal after removing comments (and trimming) result in a
// single entry.
//
// Example:
//
//	// blocked.txt contains:
//	// // known spammers
//	// spam.com
//	// malicious.org   // since 2026
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "blocked.txt",
//	    dynconfig.LoadStringLineSetFormat(dynconfig.LineFormat{
//	        CommentMarkers: []string{"//"},
//	    }),
//	    nil, nil, nil, nil,
//	)
//	// loader.Get() = map[string]struct{}{"spam.com": {}, "malicious.org": {}}
func LoadStringLineSetFormat(format LineFormat) func(file fs.File) (map[string]struct{}, error) {
	return LoadStringLineSetFormatT[string](format)
}

// LoadStringLineSetFormatT returns a load function that reads the file as a
// unique set of strings of type T, one per line, handling whitespace and line
// comments as configured by format.
//
// Type T must be a string type.
//
// Example:
//
//	type IPAddress string
//
//	loader := dynconfig.MustLoadAndWatch(
//	    "blocked-ips.txt",
//	    dynconfig.LoadStringLineSetFormatT[IPAddress](dynconfig.LineFormat{
//	        TrimSpace:      true,
//	        CommentMarkers: []string{"#"},
//	    }),
//	    nil, nil, nil, nil,
//	)
func LoadStringLineSetFormatT[T ~string](format LineFormat) func(file fs.File) (map[T]struct{}, error) {
	filter, err := newLineFilter(format)
	return func(file fs.File) (map[T]struct{}, error) {
		if err != nil {
			return nil, err
		}
		return loadLineSet[T](file, filter)
	}
}

// lineFilter is a LineFormat prepared for reading: its comment markers are
// checked and copied so the caller can not change them later.
type lineFilter struct {
	trimSpace bool
	markers   []string
}

// newLineFilter checks the comment markers of format and returns the lineFilter
// for it. An empty marker is an error because it would match at the start of
// every line and turn the whole file into a comment. A marker containing a line
// break is an error because the lines are searched after SplitLines removed the
// line breaks, so such a marker never matches and its comments would silently
// become values.
func newLineFilter(format LineFormat) (lineFilter, error) {
	for _, marker := range format.CommentMarkers {
		if marker == "" {
			return lineFilter{}, errors.New("dynconfig: empty line comment marker in LineFormat.CommentMarkers")
		}
		if strings.ContainsAny(marker, "\n\r") {
			return lineFilter{}, fmt.Errorf("dynconfig: line comment marker %q in LineFormat.CommentMarkers contains a line break", marker)
		}
	}
	return lineFilter{
		trimSpace: format.TrimSpace,
		markers:   slices.Clone(format.CommentMarkers),
	}, nil
}

// changesLines reports whether the filter changes or skips any line.
func (f lineFilter) changesLines() bool {
	return f.trimSpace || len(f.markers) > 0
}

// readLines reads the file without its byte order mark and splits it into
// lines with SplitLines. Removing the BOM first means it neither becomes part
// of the first value nor hides a comment on the first line.
func (f lineFilter) readLines(file fs.File) ([]string, error) {
	str, err := readAllStringWithoutBOM(file)
	if err != nil {
		return nil, err
	}
	return SplitLines(str), nil
}

// value returns line with its comment and whitespace removed as configured,
// and false if the line yields no value.
func (f lineFilter) value(line string) (string, bool) {
	if !f.changesLines() {
		return line, true
	}
	line = stripLineComment(line, f.markers)
	if f.trimSpace {
		line = strings.TrimSpace(line)
	}
	return line, line != ""
}

// loadLines reads the file as a slice of lines of type T, filtered by filter.
// It is shared by all line slice loaders.
func loadLines[T ~string](file fs.File, filter lineFilter) ([]T, error) {
	strs, err := filter.readLines(file)
	if err != nil {
		return nil, err
	}
	if !filter.changesLines() {
		return *(*[]T)(unsafe.Pointer(&strs)), nil //#nosec G103 -- unsafe OK
	}
	lines := make([]T, 0, len(strs))
	for _, s := range strs {
		if v, ok := filter.value(s); ok {
			lines = append(lines, T(v))
		}
	}
	return lines, nil
}

// loadLineSet reads the file as a set of lines of type T, filtered by filter.
// It is shared by all line set loaders.
func loadLineSet[T ~string](file fs.File, filter lineFilter) (map[T]struct{}, error) {
	strs, err := filter.readLines(file)
	if err != nil {
		return nil, err
	}
	set := make(map[T]struct{}, len(strs))
	for _, s := range strs {
		if v, ok := filter.value(s); ok {
			set[T(v)] = struct{}{}
		}
	}
	return set, nil
}

// stripLineComment returns line cut at the first occurrence of any of the
// markers, with the whitespace in front of the marker removed, or line
// unchanged if it contains none of them.
func stripLineComment(line string, markers []string) string {
	end := len(line)
	for _, marker := range markers {
		// Search the whole line, not line[:end]: a marker starting before end
		// may extend past it.
		if i := strings.Index(line, marker); i >= 0 && i < end {
			end = i
		}
	}
	if end == len(line) {
		return line
	}
	return strings.TrimRightFunc(line[:end], unicode.IsSpace)
}

// resolveLineSeparator returns the line separator to use when writing lines.
// With no arguments it defaults to a single newline ("\n"), otherwise the
// concatenated sep strings are used.
func resolveLineSeparator(sep []string) string {
	if len(sep) == 0 {
		return "\n"
	}
	return strings.Join(sep, "")
}

// sanitizeLine neutralizes the line separator inside a single value, so the
// value cannot be split across multiple lines when the file is read back. Every
// whole occurrence of separator is replaced with a single space, except when the
// separator is itself a space, in which case the occurrences are removed instead
// (replacing a space with a space would leave the separator intact).
//
// Only the configured separator is neutralized, not every character SplitLines
// breaks on: with a multi-character separator (for example "\r\n") a lone
// constituent character embedded in a value is left untouched, so a value
// containing just "\n" would still be split by the default SplitLines.
func sanitizeLine(line, separator string) string {
	if separator == " " {
		return strings.ReplaceAll(line, " ", "")
	}
	return strings.ReplaceAll(line, separator, " ")
}
