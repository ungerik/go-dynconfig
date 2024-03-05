package dynconfig

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ungerik/go-fs"
)

// Loader watches a file for changes and loads a configuration of type T from it
// using a load function. The configuration is reloaded on file changes.
type Loader[T any] struct {
	mtx          sync.Mutex
	file         fs.File
	load         func(fs.File) (T, error)
	onLoad       func(T) T
	onError      func(error) T
	onInvalidate func()
	unwatch      func() error
	config       T
	loaded       bool
}

// NewLoader returns a new Loader for the type T
// without loading the configuration yet.
func NewLoader[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) *Loader[T] {
	return &Loader[T]{
		file:         file,
		load:         load,
		onLoad:       onLoad,
		onError:      onError,
		onInvalidate: onInvalidate,
	}
}

// LoadAndWatch returns a new Loader for the type T
// that watches the given file for changes.
//
// The passed load function is called to load the configuration.
// onLoad, onError, and onInvalidate are optional callbacks.
//
// If the file's directory can't be watched, then an error is returned.
// No watching error is returned if the file does not exist yet,
// but file's directory exists.
// The file will then be loaded as soon as it is
// created within the watched directory.
//
// In case of an initial loading error
// the error is returned if onError is nil,
// else onError is called to handle the error and
// LoadAndWatch returns the Loader without the error.
func LoadAndWatch[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) (*Loader[T], error) {
	l := NewLoader(file, load, onLoad, onError, onInvalidate)
	err := l.Watch() // May invalidate before load which is OK
	if err != nil {
		return nil, err
	}
	_, err = l.Load()
	if err != nil && onError == nil {
		// Unwatch and return error if no onError
		return nil, errors.Join(err, l.unwatch())
	}
	// In case of an error, onError was called within Load
	return l, nil
}

// MustLoadAndWatch calls LoadAndWatch and panics on any error that it returns.
func MustLoadAndWatch[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) *Loader[T] {
	l, err := LoadAndWatch(file, load, onLoad, onError, onInvalidate)
	if err != nil {
		panic(err)
	}
	return l
}

// File returns the file that is watched for changes.
func (l *Loader[T]) File() fs.File {
	return l.file
}

// Loaded returns true if the configuration has been loaded.
func (l *Loader[T]) Loaded() bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	return l.loaded
}

// Invalidate marks the configuration as not loaded.
func (l *Loader[T]) Invalidate() {
	l.mtx.Lock()
	l.loaded = false
	l.mtx.Unlock()

	if l.onInvalidate != nil {
		l.onInvalidate()
	}
}

// Watch starts watching the file's directory for
// writes of the file to invalidate the configuration.
// A deletion of the file does not invalidate
// the configuration, but a (re)creation does.
// It returns an error if the file is already watched.
func (l *Loader[T]) Watch() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.unwatch != nil {
		return fmt.Errorf("config file already watched: %s", l.file)
	}
	unwatch, err := l.file.Dir().Watch(func(f fs.File, e fs.Event) {
		if f == l.file && (e.HasCreate() || e.HasWrite()) {
			l.Invalidate()
		}
	})
	if err != nil {
		return fmt.Errorf("watch config file error: %w", err)
	}
	l.unwatch = unwatch
	return nil
}

// Unwatch stops watching the file for changes.
// It returns an error if the file is not watched.
func (l *Loader[T]) Unwatch() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.unwatch == nil {
		return fmt.Errorf("config file not watched: %s", l.file)
	}
	err := l.unwatch()
	l.unwatch = nil
	return err
}

// Load returns the loaded configuration,
// or if not loaded or invalidated loads it first.
// In case of a loading error the last known configuration is returned,
// or whatever onError returns if onError is not nil.
func (l *Loader[T]) Load() (T, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.loaded {
		return l.config, nil
	}

	config, err := l.load(l.file)
	if err != nil {
		if l.onError != nil {
			return l.onError(err), err
		}
		return l.config, err // Return last known config
	}
	if l.onLoad != nil {
		l.config = l.onLoad(config)
	} else {
		l.config = config
	}
	l.loaded = true
	return l.config, nil
}

// Get returns the loaded configuration,
// or if not loaded or invalidated loads it first.
// In case of a loading error the last known configuration is returned,
// or whatever onError returns if onError is not nil.
func (l *Loader[T]) Get() T {
	config, _ := l.Load()
	return config
}
