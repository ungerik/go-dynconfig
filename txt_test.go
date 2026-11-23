package dynconfig

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ungerik/go-fs"
)

// host is a custom string type used to exercise the generic ~string variants.
type host string

// linesContent exercises the line-splitting behavior:
//   - mixed "\n", "\r\n" line endings
//   - a duplicate line ("alpha")
//   - an empty line (dropped by SplitLines / FieldsFunc)
//   - a whitespace-only line ("  ")
//   - a line with surrounding spaces ("  delta  ")
const linesContent = "alpha\nbeta\r\ngamma\nalpha\n\n  \n  delta  \n"

func TestLoadString(t *testing.T) {
	const content = "  hello world  \n"
	file := memFile(t, "data.txt", content)

	got, err := LoadString(file)
	if err != nil {
		t.Fatalf("LoadString: %s", err)
	}
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestLoadString_FileNotExist(t *testing.T) {
	file := missingMemFile(t, "data.txt")

	_, err := LoadString(file)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadStringT(t *testing.T) {
	const content = "  hello world  \n"
	file := memFile(t, "data.txt", content)

	got, err := LoadStringT[host](file)
	if err != nil {
		t.Fatalf("LoadStringT: %s", err)
	}
	if got != host(content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestLoadStringTrimSpace(t *testing.T) {
	file := memFile(t, "data.txt", "  hello world  \n")

	got, err := LoadStringTrimSpace(file)
	if err != nil {
		t.Fatalf("LoadStringTrimSpace: %s", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

// TestLoadString_ByteOrderMark verifies that the string loaders remove a UTF-8
// byte order mark at the start of the file. LoadString keeps the whitespace
// after it, the TrimSpace loaders trim it.
func TestLoadString_ByteOrderMark(t *testing.T) {
	file := memFile(t, "data.txt", "\uFEFF  token123  \n")

	tests := []struct {
		name string
		load func(fs.File) (string, error)
		want string
	}{
		{"LoadString", LoadString, "  token123  \n"},
		{"LoadStringT", func(f fs.File) (string, error) { s, err := LoadStringT[host](f); return string(s), err }, "  token123  \n"},
		{"LoadStringTrimSpace", LoadStringTrimSpace, "token123"},
		{"LoadStringTrimSpaceT", func(f fs.File) (string, error) { s, err := LoadStringTrimSpaceT[host](f); return string(s), err }, "token123"},
	}
	for _, tt := range tests {
		got, err := tt.load(file)
		if err != nil {
			t.Fatalf("%s: %s", tt.name, err)
		}
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLoadStringTrimSpaceT(t *testing.T) {
	file := memFile(t, "data.txt", "  secret123  \n")

	got, err := LoadStringTrimSpaceT[host](file)
	if err != nil {
		t.Fatalf("LoadStringTrimSpaceT: %s", err)
	}
	if got != host("secret123") {
		t.Errorf("got %q, want %q", got, "secret123")
	}
}

func TestLoadStringLines(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLines(file)
	if err != nil {
		t.Fatalf("LoadStringLines: %s", err)
	}
	want := []string{"alpha", "beta", "gamma", "alpha", "  ", "  delta  "}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadStringLinesT(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLinesT[host](file)
	if err != nil {
		t.Fatalf("LoadStringLinesT: %s", err)
	}
	want := []host{"alpha", "beta", "gamma", "alpha", "  ", "  delta  "}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadStringLinesTrimSpace(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLinesTrimSpace(file)
	if err != nil {
		t.Fatalf("LoadStringLinesTrimSpace: %s", err)
	}
	// Each line trimmed, empty/whitespace-only lines removed, duplicates kept.
	want := []string{"alpha", "beta", "gamma", "alpha", "delta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadStringLinesTrimSpaceT(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLinesTrimSpaceT[host](file)
	if err != nil {
		t.Fatalf("LoadStringLinesTrimSpaceT: %s", err)
	}
	want := []host{"alpha", "beta", "gamma", "alpha", "delta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadStringLineSet(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLineSet(file)
	if err != nil {
		t.Fatalf("LoadStringLineSet: %s", err)
	}
	// Untrimmed unique lines; duplicate "alpha" collapses to one key.
	want := map[string]struct{}{
		"alpha":     {},
		"beta":      {},
		"gamma":     {},
		"  ":        {},
		"  delta  ": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadStringLineSetT(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLineSetT[host](file)
	if err != nil {
		t.Fatalf("LoadStringLineSetT: %s", err)
	}
	want := map[host]struct{}{
		"alpha":     {},
		"beta":      {},
		"gamma":     {},
		"  ":        {},
		"  delta  ": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadStringLineSetTrimSpace(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLineSetTrimSpace(file)
	if err != nil {
		t.Fatalf("LoadStringLineSetTrimSpace: %s", err)
	}
	// Trimmed unique lines; whitespace-only line dropped.
	want := map[string]struct{}{
		"alpha": {},
		"beta":  {},
		"gamma": {},
		"delta": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLoadStringLineSetTrimSpaceT(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	got, err := LoadStringLineSetTrimSpaceT[host](file)
	if err != nil {
		t.Fatalf("LoadStringLineSetTrimSpaceT: %s", err)
	}
	want := map[host]struct{}{
		"alpha": {},
		"beta":  {},
		"gamma": {},
		"delta": {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// readBack reads the raw content of file, failing the test on error.
func readBack(t *testing.T, file fs.File) string {
	t.Helper()
	got, err := LoadString(file)
	if err != nil {
		t.Fatalf("read back: %s", err)
	}
	return got
}

func TestSaveString(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveString(file, "hello world"); err != nil {
		t.Fatalf("SaveString: %s", err)
	}
	if got := readBack(t, file); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestSaveStringT(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringT[host](file, "secret123"); err != nil {
		t.Fatalf("SaveStringT: %s", err)
	}
	if got := readBack(t, file); got != "secret123" {
		t.Errorf("got %q, want %q", got, "secret123")
	}
}

func TestSaveStringTrimSpace(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringTrimSpace(file, "  hello world  \n"); err != nil {
		t.Fatalf("SaveStringTrimSpace: %s", err)
	}
	if got := readBack(t, file); got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestSaveStringTrimSpaceT(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringTrimSpaceT[host](file, "  secret123  \n"); err != nil {
		t.Fatalf("SaveStringTrimSpaceT: %s", err)
	}
	if got := readBack(t, file); got != "secret123" {
		t.Errorf("got %q, want %q", got, "secret123")
	}
}

func TestSaveStringLines(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLines()(file, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("SaveStringLines: %s", err)
	}
	if got := readBack(t, file); got != "alpha\nbeta\ngamma" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta\ngamma")
	}
}

func TestSaveStringLinesCustomSeparator(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLines(", ")(file, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("SaveStringLines: %s", err)
	}
	if got := readBack(t, file); got != "alpha, beta, gamma" {
		t.Errorf("got %q, want %q", got, "alpha, beta, gamma")
	}
}

func TestSaveStringLinesT(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLinesT[host]()(file, []host{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("SaveStringLinesT: %s", err)
	}
	if got := readBack(t, file); got != "alpha\nbeta\ngamma" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta\ngamma")
	}
}

func TestSaveStringLinesTrimSpace(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLinesTrimSpace()(file, []string{"  alpha  ", "", "  ", "beta"}); err != nil {
		t.Fatalf("SaveStringLinesTrimSpace: %s", err)
	}
	if got := readBack(t, file); got != "alpha\nbeta" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta")
	}
}

func TestSaveStringLinesTrimSpaceT(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLinesTrimSpaceT[host]()(file, []host{"  alpha  ", "", "beta  "}); err != nil {
		t.Fatalf("SaveStringLinesTrimSpaceT: %s", err)
	}
	if got := readBack(t, file); got != "alpha\nbeta" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta")
	}
}

func TestSaveStringLineSet(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	set := map[string]struct{}{"gamma": {}, "alpha": {}, "beta": {}}
	if err := SaveStringLineSet()(file, set); err != nil {
		t.Fatalf("SaveStringLineSet: %s", err)
	}
	// Output is sorted for determinism regardless of map iteration order.
	if got := readBack(t, file); got != "alpha\nbeta\ngamma" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta\ngamma")
	}
}

func TestSaveStringLineSetT(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	set := map[host]struct{}{"gamma": {}, "alpha": {}, "beta": {}}
	if err := SaveStringLineSetT[host]()(file, set); err != nil {
		t.Fatalf("SaveStringLineSetT: %s", err)
	}
	if got := readBack(t, file); got != "alpha\nbeta\ngamma" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta\ngamma")
	}
}

func TestSaveStringLineSetTrimSpace(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	set := map[string]struct{}{"  gamma  ": {}, "alpha": {}, "  ": {}, "": {}}
	if err := SaveStringLineSetTrimSpace()(file, set); err != nil {
		t.Fatalf("SaveStringLineSetTrimSpace: %s", err)
	}
	if got := readBack(t, file); got != "alpha\ngamma" {
		t.Errorf("got %q, want %q", got, "alpha\ngamma")
	}
}

func TestSaveStringLineSetTrimSpaceT(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	// "beta" and "  beta  " collide after trimming and must be written once.
	set := map[host]struct{}{"beta": {}, "  beta  ": {}, "alpha": {}}
	if err := SaveStringLineSetTrimSpaceT[host]()(file, set); err != nil {
		t.Fatalf("SaveStringLineSetTrimSpaceT: %s", err)
	}
	if got := readBack(t, file); got != "alpha\nbeta" {
		t.Errorf("got %q, want %q", got, "alpha\nbeta")
	}
}

// TestSaveLoadRoundTrip verifies that saving then loading lines via the default
// newline separator returns the original slice.
func TestSaveLoadRoundTrip(t *testing.T) {
	file := memFile(t, "data.txt", "")

	want := []string{"alpha", "beta", "gamma"}
	if err := SaveStringLines()(file, want); err != nil {
		t.Fatalf("SaveStringLines: %s", err)
	}
	got, err := LoadStringLines(file)
	if err != nil {
		t.Fatalf("LoadStringLines: %s", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestSaveStringLinesSanitizesSeparator verifies that a separator character
// embedded in a line value is replaced with a space, so the value is not split
// into multiple lines when read back. With the default "\n" separator a value
// of "a\nb" must become "a b" and round-trip as a single element.
func TestSaveStringLinesSanitizesSeparator(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLines()(file, []string{"a\nb", "c"}); err != nil {
		t.Fatalf("SaveStringLines: %s", err)
	}
	if got := readBack(t, file); got != "a b\nc" {
		t.Errorf("got %q, want %q", got, "a b\nc")
	}

	// The element count must be preserved on reload (2 in, 2 out).
	got, err := LoadStringLines(file)
	if err != nil {
		t.Fatalf("LoadStringLines: %s", err)
	}
	if want := []string{"a b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip got %q, want %q", got, want)
	}
}

// TestSaveStringLinesSpaceSeparatorRemovesSpaces verifies that when the
// separator is a space, a space inside a line value is removed (not replaced
// with another space), so the value cannot merge with its neighbors.
func TestSaveStringLinesSpaceSeparatorRemovesSpaces(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLines(" ")(file, []string{"a b", "c"}); err != nil {
		t.Fatalf("SaveStringLines: %s", err)
	}
	if got := readBack(t, file); got != "ab c" {
		t.Errorf("got %q, want %q", got, "ab c")
	}
}

// TestSaveStringLinesPreservesSpacesInValue verifies that a plain space in a
// value is kept when the separator is not a space: only an actual occurrence of
// the separator is replaced. With separator ", " the space in "a b" survives,
// while the embedded ", " in "x, y" is replaced with a space.
func TestSaveStringLinesPreservesSpacesInValue(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLines(", ")(file, []string{"a b", "x, y"}); err != nil {
		t.Fatalf("SaveStringLines: %s", err)
	}
	if got := readBack(t, file); got != "a b, x y" {
		t.Errorf("got %q, want %q", got, "a b, x y")
	}
}

// TestSaveStringLinesTrimSpaceSanitizeThenTrim verifies that sanitizing happens
// before trimming: a non-whitespace separator (",") embedded at a value's
// boundary is replaced with spaces, which must then be trimmed away, yielding
// "x" rather than " x ".
func TestSaveStringLinesTrimSpaceSanitizeThenTrim(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	if err := SaveStringLinesTrimSpace(",")(file, []string{",x,", "y"}); err != nil {
		t.Fatalf("SaveStringLinesTrimSpace: %s", err)
	}
	if got := readBack(t, file); got != "x,y" {
		t.Errorf("got %q, want %q", got, "x,y")
	}
}

// TestSaveStringLineSetSanitizeCollision verifies that two members that collide
// only after separator sanitizing are written once (the sort+compact dedup).
func TestSaveStringLineSetSanitizeCollision(t *testing.T) {
	file := memFile(t, "data.txt", "old")

	// "a\nb" sanitizes to "a b", colliding with the literal "a b".
	set := map[string]struct{}{"a\nb": {}, "a b": {}, "z": {}}
	if err := SaveStringLineSet()(file, set); err != nil {
		t.Fatalf("SaveStringLineSet: %s", err)
	}
	if got := readBack(t, file); got != "a b\nz" {
		t.Errorf("got %q, want %q", got, "a b\nz")
	}
}

// TestSanitizeLine exercises sanitizeLine directly across the separator variants
// it must handle, including the documented limitation that only the whole
// configured separator is neutralized.
func TestSanitizeLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		separator string
		want      string
	}{
		{"no separator unchanged", "abc", "\n", "abc"},
		{"newline replaced with space", "a\nb", "\n", "a b"},
		{"multiple newlines", "a\nb\nc", "\n", "a b c"},
		{"newline at boundaries", "\nx\n", "\n", " x "},
		{"plain spaces preserved", "a b", "\n", "a b"},
		{"empty line", "", "\n", ""},
		{"space separator drops spaces", "a b c", " ", "abc"},
		{"space separator no spaces unchanged", "abc", " ", "abc"},
		{"space separator collapses runs", "  a  b  ", " ", "ab"},
		{"multi-char separator replaced", "x, y", ", ", "x y"},
		{"multi-char separator partial match untouched", "c,d", ", ", "c,d"},
		{"crlf separator replaced", "a\r\nb", "\r\n", "a b"},
		{"crlf separator lone newline untouched", "a\nb", "\r\n", "a\nb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeLine(tt.line, tt.separator); got != tt.want {
				t.Errorf("sanitizeLine(%q, %q) = %q, want %q", tt.line, tt.separator, got, tt.want)
			}
		})
	}
}

// TestResolveLineSeparator verifies the default and the concatenation of the
// variadic separator arguments.
func TestResolveLineSeparator(t *testing.T) {
	tests := []struct {
		name string
		sep  []string
		want string
	}{
		{"nil defaults to newline", nil, "\n"},
		{"empty slice defaults to newline", []string{}, "\n"},
		{"single separator", []string{"\r\n"}, "\r\n"},
		{"comma space separator", []string{", "}, ", "},
		{"multiple args concatenated", []string{"a", "b"}, "ab"},
		{"single empty string", []string{""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveLineSeparator(tt.sep); got != tt.want {
				t.Errorf("resolveLineSeparator(%q) = %q, want %q", tt.sep, got, tt.want)
			}
		})
	}
}

// commentedContent exercises the LineFormat comment handling:
//   - a full-line "#" comment (skipped)
//   - a value with a trailing "#" comment ("alpha  # first", yields "alpha"
//     because the whitespace in front of a comment is removed too)
//   - a full-line "//" comment (only a comment with the "//" marker)
//   - a value with a trailing "//" comment ("beta  // second")
//   - a duplicate value ("alpha")
//   - an indented comment ("  # indented", skipped also without TrimSpace)
//   - a value with surrounding spaces and no comment ("  gamma  ", kept as is
//     without TrimSpace)
const commentedContent = "# header\nalpha  # first\n// note\nbeta  // second\nalpha\n  # indented\n  gamma  \n"

var (
	hashComments          = LineFormat{CommentMarkers: []string{"#"}}
	hashSlashComments     = LineFormat{CommentMarkers: []string{"#", "//"}}
	trimHashComments      = LineFormat{TrimSpace: true, CommentMarkers: []string{"#"}}
	trimHashSlashComments = LineFormat{TrimSpace: true, CommentMarkers: []string{"#", "//"}}
)

func TestLoadStringLinesFormat(t *testing.T) {
	file := memFile(t, "data.txt", commentedContent)

	tests := []struct {
		name   string
		format LineFormat
		want   []string
	}{
		{"# markers", hashComments, []string{"alpha", "// note", "beta  // second", "alpha", "  gamma  "}},
		{"# and // markers", hashSlashComments, []string{"alpha", "beta", "alpha", "  gamma  "}},
		{"trim, # markers", trimHashComments, []string{"alpha", "// note", "beta  // second", "alpha", "gamma"}},
		{"trim, # and // markers", trimHashSlashComments, []string{"alpha", "beta", "alpha", "gamma"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadStringLinesFormat(tt.format)(file)
			if err != nil {
				t.Fatalf("LoadStringLinesFormat: %s", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadStringLinesFormatT(t *testing.T) {
	file := memFile(t, "data.txt", commentedContent)

	got, err := LoadStringLinesFormatT[host](hashSlashComments)(file)
	if err != nil {
		t.Fatalf("LoadStringLinesFormatT: %s", err)
	}
	want := []host{"alpha", "beta", "alpha", "  gamma  "}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoadStringLineSetFormat(t *testing.T) {
	file := memFile(t, "data.txt", commentedContent)

	tests := []struct {
		name   string
		format LineFormat
		want   map[string]struct{}
	}{
		// "alpha  # first" and "alpha" collide because the whitespace in
		// front of the comment is removed.
		{"# and // markers", hashSlashComments, map[string]struct{}{"alpha": {}, "beta": {}, "  gamma  ": {}}},
		{"trim, # and // markers", trimHashSlashComments, map[string]struct{}{"alpha": {}, "beta": {}, "gamma": {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadStringLineSetFormat(tt.format)(file)
			if err != nil {
				t.Fatalf("LoadStringLineSetFormat: %s", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadStringLineSetFormatT(t *testing.T) {
	file := memFile(t, "data.txt", commentedContent)

	got, err := LoadStringLineSetFormatT[host](trimHashComments)(file)
	if err != nil {
		t.Fatalf("LoadStringLineSetFormatT: %s", err)
	}
	want := map[host]struct{}{"alpha": {}, "// note": {}, "beta  // second": {}, "gamma": {}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestLoadStringLinesFormat_ZeroValue verifies that the zero LineFormat and a
// TrimSpace-only LineFormat read the lines like the plain loaders.
func TestLoadStringLinesFormat_ZeroValue(t *testing.T) {
	file := memFile(t, "data.txt", linesContent)

	for _, tt := range []struct {
		format    LineFormat
		wantLines func(fs.File) ([]string, error)
		wantSet   func(fs.File) (map[string]struct{}, error)
	}{
		{LineFormat{}, LoadStringLines, LoadStringLineSet},
		{LineFormat{TrimSpace: true}, LoadStringLinesTrimSpace, LoadStringLineSetTrimSpace},
	} {
		t.Run(fmt.Sprintf("TrimSpace=%t", tt.format.TrimSpace), func(t *testing.T) {
			lines, err := LoadStringLinesFormat(tt.format)(file)
			if err != nil {
				t.Fatalf("LoadStringLinesFormat: %s", err)
			}
			wantLines, _ := tt.wantLines(file)
			if !reflect.DeepEqual(lines, wantLines) {
				t.Errorf("lines: got %q, want %q", lines, wantLines)
			}

			set, err := LoadStringLineSetFormat(tt.format)(file)
			if err != nil {
				t.Fatalf("LoadStringLineSetFormat: %s", err)
			}
			wantSet, _ := tt.wantSet(file)
			if !reflect.DeepEqual(set, wantSet) {
				t.Errorf("set: got %q, want %q", set, wantSet)
			}
		})
	}
}

// TestLoadStringLines_ByteOrderMark verifies that the plain line loaders
// remove a UTF-8 byte order mark at the start of the file instead of keeping
// it in the first value.
func TestLoadStringLines_ByteOrderMark(t *testing.T) {
	file := memFile(t, "data.txt", "\uFEFFapi.example.com\ndb.example.com\n")

	wantLines := []string{"api.example.com", "db.example.com"}
	for name, load := range map[string]func(fs.File) ([]string, error){
		"LoadStringLines":          LoadStringLines,
		"LoadStringLinesTrimSpace": LoadStringLinesTrimSpace,
		"LoadStringLinesFormat":    LoadStringLinesFormat(LineFormat{}),
	} {
		got, err := load(file)
		if err != nil {
			t.Fatalf("%s: %s", name, err)
		}
		if !reflect.DeepEqual(got, wantLines) {
			t.Errorf("%s: got %q, want %q", name, got, wantLines)
		}
	}

	wantSet := map[string]struct{}{"api.example.com": {}, "db.example.com": {}}
	for name, load := range map[string]func(fs.File) (map[string]struct{}, error){
		"LoadStringLineSet":          LoadStringLineSet,
		"LoadStringLineSetTrimSpace": LoadStringLineSetTrimSpace,
		"LoadStringLineSetFormat":    LoadStringLineSetFormat(LineFormat{}),
	} {
		got, err := load(file)
		if err != nil {
			t.Fatalf("%s: %s", name, err)
		}
		if !reflect.DeepEqual(got, wantSet) {
			t.Errorf("%s: got %q, want %q", name, got, wantSet)
		}
	}
}

// TestLoadStringLinesFormat_ByteOrderMark verifies that a UTF-8 byte order
// mark at the start of the file does not hide a comment on the first line.
func TestLoadStringLinesFormat_ByteOrderMark(t *testing.T) {
	file := memFile(t, "data.txt", "\uFEFF# allowed hosts\napi.example.com\n")

	for _, format := range []LineFormat{hashComments, trimHashComments} {
		got, err := LoadStringLinesFormat(format)(file)
		if err != nil {
			t.Fatalf("LoadStringLinesFormat: %s", err)
		}
		if want := []string{"api.example.com"}; !reflect.DeepEqual(got, want) {
			t.Errorf("TrimSpace=%t: got %q, want %q", format.TrimSpace, got, want)
		}
	}
}

// TestLoadStringLinesFormat_CustomSplitLines verifies that lines that are
// empty after removing a comment are skipped, while the plain loaders keep the
// empty lines a custom SplitLines emits.
func TestLoadStringLinesFormat_CustomSplitLines(t *testing.T) {
	defaultSplitLines := SplitLines
	t.Cleanup(func() { SplitLines = defaultSplitLines })
	SplitLines = func(str string) []string { return strings.Split(str, "\n") }

	file := memFile(t, "data.txt", "a\n\n# comment\nb")

	got, err := LoadStringLinesFormat(hashComments)(file)
	if err != nil {
		t.Fatalf("LoadStringLinesFormat: %s", err)
	}
	if want := []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("LoadStringLinesFormat: got %q, want %q", got, want)
	}

	got, err = LoadStringLines(file)
	if err != nil {
		t.Fatalf("LoadStringLines: %s", err)
	}
	if want := []string{"a", "", "# comment", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("LoadStringLines: got %q, want %q", got, want)
	}
}

// TestLoadStringLinesFormat_EmptyMarker verifies that an empty comment marker
// is returned as an error by the load function instead of panicking.
func TestLoadStringLinesFormat_EmptyMarker(t *testing.T) {
	file := memFile(t, "data.txt", commentedContent)

	for _, markers := range [][]string{{""}, {"#", ""}} {
		format := LineFormat{CommentMarkers: markers}
		if _, err := LoadStringLinesFormat(format)(file); err == nil {
			t.Errorf("LoadStringLinesFormat(%q): expected error", markers)
		}
		if _, err := LoadStringLineSetFormat(format)(file); err == nil {
			t.Errorf("LoadStringLineSetFormat(%q): expected error", markers)
		}
	}
}

// TestLoadStringLinesFormat_MarkersCopied verifies that changing the caller's
// CommentMarkers slice after creating the load function has no effect.
func TestLoadStringLinesFormat_MarkersCopied(t *testing.T) {
	file := memFile(t, "data.txt", "a # comment\n")

	markers := []string{"#"}
	load := LoadStringLinesFormat(LineFormat{CommentMarkers: markers})
	markers[0] = ";"

	got, err := load(file)
	if err != nil {
		t.Fatalf("LoadStringLinesFormat: %s", err)
	}
	if want := []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestLoadStringLinesFormat_FileNotExist verifies that the LineFormat loaders
// return the read error of a missing file.
func TestLoadStringLinesFormat_FileNotExist(t *testing.T) {
	file := missingMemFile(t, "data.txt")

	if _, err := LoadStringLinesFormat(hashComments)(file); err == nil {
		t.Error("LoadStringLinesFormat: expected error for missing file")
	}
	if _, err := LoadStringLineSetFormatT[host](trimHashComments)(file); err == nil {
		t.Error("LoadStringLineSetFormatT: expected error for missing file")
	}
}

// TestLoadStringLinesFormat_SaveDropsComments documents the known limitation
// that writing a config back with a save function drops the comments of the
// file, because the loaded value does not carry them.
func TestLoadStringLinesFormat_SaveDropsComments(t *testing.T) {
	file := memFile(t, "data.txt", "# allowed hosts\nb.example.com # backup\na.example.com\n")
	loader := NewLoader(
		file,
		LoadStringLineSetFormat(trimHashComments),
		SaveStringLineSetTrimSpace(),
		nil, nil, nil,
	)

	err := loader.Mutate(false, func(set map[string]struct{}) (map[string]struct{}, error) {
		set["c.example.com"] = struct{}{}
		return set, nil
	})
	if err != nil {
		t.Fatalf("Mutate: %s", err)
	}
	if got, want := readBack(t, file), "a.example.com\nb.example.com\nc.example.com"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripLineComment(t *testing.T) {
	tests := []struct {
		line    string
		markers []string
		want    string
	}{
		{"value", []string{"#"}, "value"},
		{"  value  ", []string{"#"}, "  value  "},
		{"value # comment", []string{"#"}, "value"},
		{"  value\t# comment", []string{"#"}, "  value"},
		{"# comment", []string{"#"}, ""},
		{"  # comment", []string{"#"}, ""},
		{"a # b # c", []string{"#"}, "a"},
		{"a // b # c", []string{"#", "//"}, "a"},
		{"a # b // c", []string{"#", "//"}, "a"},
		{"https://example.com", []string{"//"}, "https:"},
		{"a; b", []string{"#", ";"}, "a"},
		// "/#" starts before "#", so it must win even though "#" is listed first.
		{"a/#b", []string{"#", "/#"}, "a"},
		{"", []string{"#"}, ""},
		{"value", nil, "value"},
	}
	for _, tt := range tests {
		if got := stripLineComment(tt.line, tt.markers); got != tt.want {
			t.Errorf("stripLineComment(%q, %q) = %q, want %q", tt.line, tt.markers, got, tt.want)
		}
	}
}
