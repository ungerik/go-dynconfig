module github.com/ungerik/go-dynconfig/example

go 1.26.0

replace github.com/ungerik/go-dynconfig => ../

replace github.com/ungerik/go-dynconfig/loadenv => ../loadenv

// Replaced with the local version of the packages
require (
	github.com/ungerik/go-dynconfig v0.0.0-00010101000000-000000000000
	github.com/ungerik/go-dynconfig/loadenv v0.0.0-00010101000000-000000000000
)

require (
	github.com/caarlos0/env/v7 v7.1.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/pkg/xattr v0.4.12 // indirect
	github.com/ungerik/go-fs v1.0.0-beta.2 // indirect
	golang.org/x/sys v0.49.0 // indirect
)
