# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Go module versioning](https://go.dev/doc/modules/version-numbers)
(`vMAJOR.MINOR.PATCH`).

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
