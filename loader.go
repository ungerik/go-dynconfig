package dynconfig

import (
	"sync"

	"github.com/ungerik/go-fs"
)

type Loader[T any] struct {
	mtx          sync.Mutex
	file         fs.File
	load         func(fs.File) (T, error)
	onLoad       func(T) T
	onError      func(error) T
	onInvalidate func()
	config       T
	loaded       bool
}

func New[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) (*Loader[T], error) {
	l := &Loader[T]{
		file:         file,
		load:         load,
		onLoad:       onLoad,
		onError:      onError,
		onInvalidate: onInvalidate,
	}
	err := file.Dir().Watch(func(f fs.File, e fs.Event) {
		if f == file && e == fs.EventCreate || e == fs.EventWrite {
			l.Invalidate()
		}
	})
	if err != nil {
		return nil, err
	}
	l.Get()
	return l, nil
}

func MustNew[T any](file fs.File, load func(fs.File) (T, error), onLoad func(T) T, onError func(error) T, onInvalidate func()) *Loader[T] {
	l, err := New(file, load, onLoad, onError, onInvalidate)
	if err != nil {
		panic(err)
	}
	return l
}

func (l *Loader[T]) Invalidate() {
	l.mtx.Lock()
	l.loaded = false
	l.mtx.Unlock()
	if l.onInvalidate != nil {
		l.onInvalidate()
	}
}

func (l *Loader[T]) Get() T {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if l.loaded {
		return l.config
	}

	config, err := l.load(l.file)
	if err != nil {
		if l.onError != nil {
			return l.onError(err)
		}
		return l.config // Return last known config
	}
	if l.onLoad != nil {
		l.config = l.onLoad(config)
	} else {
		l.config = config
	}
	l.loaded = true
	return l.config
}
