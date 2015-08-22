package cleanup

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
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

func registerFuncs() {
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

func TestRun(t *testing.T) {
	registerFuncs()

	wg, ch := catch(os.Interrupt, syscall.SIGTERM)

	ch <- syscall.SIGTERM

	wg.Wait()
	if n := atomic.LoadInt32(&x); n != sum(1, incs) {
		t.Error("Not all cleanup functions ran!")
	}
}

func catch(signals ...os.Signal) (*sync.WaitGroup, chan os.Signal) {
	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	go func() {
		select {
		case <-ch:
			Run()
			wg.Done()
		}
	}()

	return &wg, ch
}
