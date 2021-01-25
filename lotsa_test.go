package jj_test

import (
	"fmt"
	"github.com/bingoohuang/jj"
	"os"
	"sync/atomic"
	"testing"
)

func ExampleOps() {
	{
		var total int64
		jj.Ops(1000000, 4,
			func(i, thread int) {
				atomic.AddInt64(&total, 1)
			},
		)
		fmt.Println(total)
	}

	{
		var total int64
		// To output some benchmarking results, set the jj.OpsOutput prior to calling jj.Ops
		jj.OpsOutput = os.Stdout
		jj.Ops(1000000, 4,
			func(i, thread int) {
				atomic.AddInt64(&total, 1)
			},
		)
	}

	// OpsOutput:
	// 1000000
	// 1,000,000 ops over 4 threads in 23ms, 43,965,515/sec, 22 ns/op
}

func TestOps(t *testing.T) {
	var threads = 4
	var N = threads * 10000
	var total int64
	jj.Ops(N, threads, func(i, thread int) {
		atomic.AddInt64(&total, 1)
	})
	if total != int64(N) {
		t.Fatal("invalid total")
	}
}
