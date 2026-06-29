module github.com/ungerik/go-dynconfig/example

go 1.25.0

replace github.com/ungerik/go-dynconfig => ../

replace github.com/ungerik/go-dynconfig/loadenv => ../loadenv

// Replaced with the local version of the packages
require (
	github.com/ungerik/go-dynconfig v0.0.0-00010101000000-000000000000
	github.com/ungerik/go-dynconfig/loadenv v0.0.0-00010101000000-000000000000
)

require (
	github.com/caarlos0/env/v7 v7.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/pkg/xattr v0.4.12 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/ungerik/go-fs v0.0.0-20260629070125-ad84dc607eca // indirect
	golang.org/x/sys v0.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
