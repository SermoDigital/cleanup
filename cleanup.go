// Copyright (c) 2015 Sermo Digital, LLC.
// Unauthorized copying of this file via any medium is strictly prohibited

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

var (
	wg       sync.WaitGroup
	funcMap  = make(map[string]interface{})
	funcArgs = make(map[string][]interface{})
	funcMu   = &sync.Mutex{}
)

// Register registers a function with the given name and arguments to be
// called when the server exits.
func Register(name string, fn interface{}, args ...interface{}) {
	if _, ok := funcMap[name]; ok {
		glog.Fatalf("Unable to re-register function %v under name %q", fn, name)
	}

	funcMu.Lock()
	funcMap[name] = fn
	funcArgs[name] = args
	funcMu.Unlock()
}

// Run invokes the given function and arguments for each value in
// the map.
func Run() {
	for key, val := range funcMap {
		args := funcArgs[key]
		call(val, args...)
	}
}

// Catch catches a varying amount of signals and returns a pointer to the
// WaitGroup that should be used to block the return of your main thread.
func Catch(signals ...os.Signal) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	go func() {
		select {
		case s := <-ch:
			fmt.Println(s)
			Run()
			wg.Done()
		}
	}()

	return &wg
}

// RunAndQuit does just that -- calls Run and then calls os.Exit with the
// given signal code.
func RunAndQuit(s os.Signal) {
	Run()
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
