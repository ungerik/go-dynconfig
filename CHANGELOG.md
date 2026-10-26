# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Go module versioning](https://go.dev/doc/modules/version-numbers)
(`vMAJOR.MINOR.PATCH`).

## [Unreleased]

### Changed

- **Breaking:** Requires `ungerik/go-fs` v1.0.0-rc.1, so modules that use this
  package build with it too. Code that calls go-fs directly has to follow its
  renames: `File.URL` is `File.URI`, `fs.FileURLs` is `fs.FileURIs`,
  `fs.ReadHeader` and `fs.ReadHeaderString` take a `ctx` first, and
  `fsimpl.ReadonlyFileBuffer` is `ReadOnlyFileBuffer`. File URIs follow
  RFC 8089: `file://localhost/dir/config.json` is the absolute path
  `/dir/config.json`, and a relative URI like `file://conf/app.json` names the
  host `conf`, which is the UNC path `\\conf\app.json` on Windows and fails to
  load, watch and save elsewhere. See the go-fs CHANGELOG for all changes.
- **Breaking:** With go-fs v1.0.0-rc.1, an empty indent passed to
  `SaveJSONIndent[T]` or `SaveXMLIndent[T]` puts every element on its own line
  without indentation instead of writing compact output. Use `SaveJSON[T]` or
  `SaveXML[T]` for compact output. With any indent, `LoadXML` reads the added
  whitespace back as character data into `,chardata` and `,innerxml` fields,
  so the text of a `,chardata` field grows with every save and load cycle.
- `NewLoader`, and so `LoadAndWatch` and `MustLoadAndWatch`, resolve a relative
  local path to an absolute path when the `Loader` is created. `Loader.File`
  returns it and the load and save functions are called with it. The `Loader`
  keeps watching, loading and saving the same file after `os.Chdir` or after
  its working directory was removed; before, a reload after `os.Chdir` read a
  file with the same name in the new working directory.

### Fixed

- `Set` and `Mutate` create the temporary file of the atomic write in the
  directory of a relative config path like `config.json` instead of
  `os.TempDir()`. The rename failed when the temporary directory was on another
  file system, and the config was copied into the shared temporary directory.
- `Watch`, and so `LoadAndWatch`, reloads a config whose path the file system
  watcher reports in another form: a relative path like `config.json` never
  reloaded on Linux, `./config.json` never reloaded on macOS, and a `file://`
  URI never reloaded on either.

## [v1.1.0-beta.1] - 2026-10-24

### Added

- `SaveJSONIndent[T](indent string)` and `SaveXMLIndent[T](indent string)` return
  save functions with configurable indentation. An empty string writes compact
  output.

### Changed

- **Breaking:** `SaveJSON[T]` and `SaveXML[T]` now write compact output directly
  and can be passed to `NewLoader` without calling a factory. Call the new
  `SaveJSONIndent[T]` or `SaveXMLIndent[T]` factory for indented output.
- Raised the minimum Go version to 1.26 for `ungerik/go-fs` v1.0.0-beta.2 and
  updated `golang.org/x/sys` to v0.49.0. The dependency update also removes
  `testify` from the indirect dependencies.

## [v1.0.0] - 2026-06-29

First tagged release. Changes below are relative to the last untagged commit on `main`.

### Added

- New `loadenv` submodule (`github.com/ungerik/go-dynconfig/loadenv`) holding the
  environment-variable loaders `LoadEnvJSON`, `LoadEnvXML`, and the customizable
  `ParseEnv`. Isolating them in their own module keeps `caarlos0/env/v7` out of
  the root module's dependency tree.
- Writable configuration via two new `Loader[T]` methods:
  - `Set(config T) error` — atomically replace the entire file contents with a
    complete value you already hold.
  - `Mutate(reload bool, mutate func(config T) (T, error)) error` — atomic
    read-modify-write. With `reload` false it may reuse a valid cached value as
    the read; with `reload` true it always reads the current on-disk value first,
    making it lost-update-free even when the cache is stale.
- Atomic, cross-process-safe writes for files on the local file system: an
  exclusive OS-level advisory lock (`flock`) on the parent directory is held for
  the whole read-modify-write cycle, and the new value is written to a temporary
  file and atomically renamed over the target, so readers never observe a
  partial write and a crash leaves the original file intact. Non-local file
  systems and platforms without `flock` fall back to an in-place overwrite
  guarded by the in-process mutex (`flock_unix.go`, `flock_other.go`).
- `Save*` write-back functions, mirroring the existing `Load*` loaders, for use
  as the new `save` argument and standalone:
  - `SaveJSON`, `SaveXML` (both accept an optional indent).
  - `SaveString`, `SaveStringT`, `SaveStringTrimSpace`, `SaveStringTrimSpaceT`.
  - `SaveStringLines`, `SaveStringLinesT`, `SaveStringLinesTrimSpace`,
    `SaveStringLinesTrimSpaceT` (all accept an optional line separator).
  - `SaveStringLineSet`, `SaveStringLineSetT`, `SaveStringLineSetTrimSpace`,
    `SaveStringLineSetTrimSpaceT` (all accept an optional line separator).
- Test suite covering loaders, savers, the read-modify-write methods, and an
  in-memory file system (`loader_test.go`, `json_test.go`, `xml_test.go`,
  `txt_test.go`, `memfs_test.go`).

### Changed

- **Breaking:** the repository is now a multi-module Go workspace (`go.work`)
  with three modules: the root, `loadenv`, and `example`. Environment-variable
  support moved out of the root package — import `LoadEnvJSON`, `LoadEnvXML`, and
  `ParseEnv` from `github.com/ungerik/go-dynconfig/loadenv` instead of the root
  `dynconfig` package.
- **Breaking:** the root module no longer depends on `caarlos0/env/v7`; that
  dependency now lives only in the `loadenv` submodule. The root module's only
  direct dependencies are `ungerik/go-fs` and `golang.org/x/sys`.
- The `example/` directory became its own module
  (`github.com/ungerik/go-dynconfig/example`), wired to the local root and
  `loadenv` through `replace` directives.
- **Breaking:** `NewLoader`, `LoadAndWatch`, and `MustLoadAndWatch` now take a
  `save func(fs.File, T) error` argument (between `load` and `onLoad`). Pass
  `nil` to keep a read-only loader; the new write methods require a non-nil
  `save`.
- `golang.org/x/sys` is now a direct dependency (used for `flock(2)` on Unix).
- Bumped the Go directive to 1.25; updated `go-fs` and `fsnotify`; added
  `pkg/xattr` (indirect).
- Expanded README with atomic-mutation, write-back, and `Save*` documentation.

[v1.0.0]: https://github.com/ungerik/go-dynconfig/releases/tag/v1.0.0
