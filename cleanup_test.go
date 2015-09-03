// Copyright 2015 Sermo Digital, LLC. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package cleanup

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

var x int32

func set()        { atomic.AddInt32(&x, 1) }
func add(n int32) { atomic.AddInt32(&x, n) }

var incs = []int32{
	1,
	2,
	10,
	100,
	1000,
}

func init() {
	Register("set", set)

	Register("add1", add, incs[0])
	Register("add2", add, incs[1])
	Register("add10", add, incs[2])
	Register("add100", add, incs[3])
	Register("add1000", add, incs[4])
}

func sum(init int32, ints []int32) int32 {
	for _, v := range ints {
		init += v
	}
	return init
}

func TestThatItWorks(t *testing.T) {

	ch := make(chan os.Signal)

	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				ch <- syscall.SIGTERM
			}
		}
	}()

	wait(ch, os.Interrupt, syscall.SIGTERM)

	if n := atomic.LoadInt32(&x); n != sum(1, incs) {
		t.Error("Not all cleanup functions ran!")
	}
}

var zero = syscall.Signal(0)

func wait(ch chan os.Signal, signals ...os.Signal) {
	cfg.Do(func() {
		signal.Notify(ch, signals...)
		<-ch
		run(zero)
		close(ch)
	})
}
