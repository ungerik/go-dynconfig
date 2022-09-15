package dynconfig

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/ungerik/go-fs"
)

type Loader[T any] struct {
	mtx          sync.Mutex
	file         fs.File
	onLoad       func(T) T
	onError      func(error) T
	onInvalidate func()
	config       T
	loaded       bool
}

func New[T any](file fs.File, onLoad func(T) T, onError func(error) T, onInvalidate func()) (*Loader[T], error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	switch file.ExtLower() {
	case ".json":
		if t.Kind() != reflect.Struct && t.Kind() != reflect.Slice {
			return nil, fmt.Errorf("config type must be a struct or slice for .xml file, but is: %s", file)
		}
	case ".xml":
		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("config type must be a struct for .xml file, but is: %s", file)
		}
	case ".txt":
		if t != reflect.TypeOf("") && t != reflect.TypeOf([]string(nil)) {
			return nil, fmt.Errorf("config type must be string or []string for .txt file, but is: %s", file)
		}
	default:
		return nil, fmt.Errorf("file extension is not .json, .xml, or .txt: %s", file)
	}

	l := &Loader[T]{
		file:         file,
		onLoad:       onLoad,
		onError:      onError,
		onInvalidate: onInvalidate,
	}
	file.Dir().Watch(func(f fs.File, e fs.Event) {
		if f == file && e == fs.EventCreate || e == fs.EventWrite {
			l.Invalidate()
		}
	})
	l.Get()
	return l, nil
}

func MustNew[T any](file fs.File, onLoad func(T) T, onError func(error) T, onInvalidate func()) *Loader[T] {
	l, err := New(file, onLoad, onError, onInvalidate)
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

	var err error
	switch l.file.ExtLower() {
	case ".json":
		err = l.file.ReadJSON(&l.config)
	case ".xml":
		err = l.file.ReadXML(&l.config)
	case ".txt":
		var str string
		str, err = l.file.ReadAllString()
		if err == nil {
			switch ptr := any(&l.config).(type) {
			case *string:
				*ptr = str
			case *[]string:
				*ptr = strings.FieldsFunc(str, func(c rune) bool { return c == '\n' || c == '\r' })
			}
		}
	default:
		panic("can't happen")
	}
	if err != nil {
		if l.onError != nil {
			return l.onError(err)
		}
		return l.config
	}

	if l.onLoad != nil {
		l.config = l.onLoad(l.config)
	}
	l.loaded = true
	return l.config
}
