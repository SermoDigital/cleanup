// Copyright (c) 2015 Sermo Digital, LLcfg.
// Unauthorized copying of this file via any medium is strictly prohibited

// Package cleanup allows you to to run functions when your program exits.
package cleanup

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/golang/glog"
)

var cfg = struct {
	funcMap  map[string]interface{}
	funcArgs map[string][]interface{}
	*sync.Mutex
	*sync.Once
}{
	funcMap:  make(map[string]interface{}),
	funcArgs: make(map[string][]interface{}),
	Mutex:    &sync.Mutex{},
	Once:     &sync.Once{},
}

// Register registers a function with the given name and arguments to be
// called when the server exits. It will panic if it finds duplicate
// functions. I.e., Register("foo", fn) ... Register("foo", fn2) will panic.
func Register(name string, fn interface{}, args ...interface{}) {
	if _, ok := cfg.funcMap[name]; ok {
		glog.Fatalf("Unable to re-register function %v under name %q", fn, name)
	}

	cfg.Lock()
	cfg.funcMap[name] = fn
	cfg.funcArgs[name] = args
	cfg.Unlock()
}

// Wait catches signals and waits until all cleanup functions have has been
// ran before it returns. It will only run once, no matter how many times it's
// called.
func Wait(signals ...os.Signal) {
	cfg.Do(func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, signals...)
		run(<-ch)
		close(ch)
	})
}

// run invokes the given function and arguments for each value in
// the map.
func run(s os.Signal) {
	for key, val := range cfg.funcMap {
		args := cfg.funcArgs[key]
		call(val, args...)
	}

	sig := s.(syscall.Signal)
	os.Exit(int(sig))
}

// call calls fn with the given args. It's mostly borrowed from
// https://golang.org/src/text/template/funcs.go, with some minor
// changes.
func call(fn interface{}, args ...interface{}) (interface{}, error) {
	v := reflect.ValueOf(fn)
	typ := v.Type()

	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("non-function of type %s", typ)
	}

	numIn := typ.NumIn()

	var dddType reflect.Type

	if typ.IsVariadic() {
		if len(args) < numIn-1 {
			return nil, fmt.Errorf("wrong number of args: got %d want at least %d", len(args), numIn-1)
		}
		dddType = typ.In(numIn - 1).Elem()
	} else {
		if len(args) != numIn {
			return nil, fmt.Errorf("wrong number of args: got %d want %d", len(args), numIn)
		}
	}

	argv := make([]reflect.Value, len(args))
	for i, arg := range args {
		value := reflect.ValueOf(arg)

		var argType reflect.Type
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		} else {
			argType = dddType
		}
		if !value.IsValid() && canBeNil(argType) {
			value = reflect.Zero(argType)
		}
		if !value.Type().AssignableTo(argType) {
			return nil, fmt.Errorf("arg %d has type %s; should be %s", i, value.Type(), argType)
		}
		argv[i] = value
	}

	return v.Call(argv), nil
}

// canBeNil reports whether an untyped nil can be assigned to the type. See reflect.Zero.
func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	}
	return false
}
