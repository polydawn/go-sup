package latch

import (
	"encoding/json"
	"testing"
)

/*
	RESULTS

		BenchmarkLatchAllocation                 5000000               344 ns/op
		BenchmarkBaseline_JsonUnmarshalling       100000             14420 ns/op
		BenchmarkLatchTrigger_0Listeners        10000000               188 ns/op
		BenchmarkLatchTrigger_1Listeners         5000000               255 ns/op
		BenchmarkLatchTrigger_2Listeners         5000000               316 ns/op
		BenchmarkLatchTrigger_4Listeners         3000000               462 ns/op
		BenchmarkLatchTrigger_8Listeners         2000000               729 ns/op

	Cautions:

		- The `BenchmarkLatchTrigger_*Listeners` family uses unbuffered channels,
		  because we don't want to start measuring the vageries of goroutine scheduling.

	Observations:

		- Setting one listener costs about half as much as a small json parse.
*/

func BenchmarkLatchAllocation(b *testing.B) {
	/*
		A quick word about the cost of allocations in microbenchmarks like this:
		THEY MATTER.

		There are approximately three ways you can write this:

			1. `malloc := make([]Latch, b.N); b.ResetTimer()`
			2. that, but skip the reset
			3. `_ = New()`
			4. `var x Latch`, assign in the loop, and then sink the "unused" into `_ = x`

		On my machine:

			1. 348 ns
			2. 359 ns -- about 3%; small, but consistent
			3. 142 ns -- the compiler is optimizing things out!
			4. 326 ns -- about 9% faster than discounted prealloc; just indexing in costs that much.

		So!  While 1 and 4 are valid, 2 and 3 are *not*; and if you're
		building benchmark functions to compare against each other, they
		must consistently choose either strategy 1 or consistently strategy 4,
		or they will not be comparable apples-to-apples.

		Strategy 1 is recommended by default, because if you need to allocate
		coordinated sets of things up front, it keeps working (4 doesn't).

		Note that `StopTimer` and `StartTimer` can NOT solve these issues
		unless they're well above a certain timescale, and even then are
		rather remarkably costly in terms of wall clock run time your
		benchmark will now require.  If this `x = New()` is flanked in
		start/stop, the benchmark *still* reports 63.4 ns -- hugely out of
		line; removing the loop body results in 0.94 ns (as a no-op should)!
		Therefore, removing things from the loop body entirely remains
		your only safe option for anything measured in nanos.
		Meanwhile, running with start/stops makes wall clock timme jump from
		2sec to over 100 sec (!), because of the overhead the benchmark system
		sinks into gathering memory stats in every toggle.
	*/
	latchPool := make([]Latch, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		latchPool[i] = New()
	}
}

// Totally unrelated.  Just to put things in context.
func BenchmarkBaseline_JsonUnmarshalling(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var x interface{}
		json.Unmarshal([]byte(`{"jsonmsg":{"baseline":42}}`), x)
	}
}

func BenchmarkLatchTrigger_0Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 0) }
func BenchmarkLatchTrigger_1Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 1) }
func BenchmarkLatchTrigger_2Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 2) }
func BenchmarkLatchTrigger_4Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 4) }
func BenchmarkLatchTrigger_8Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 8) }

/*
	Target: the cost of *triggering*.

	Not:
		- allocating the latch
		- allocating the gather chans
		- signing up the gather chans
*/
func DoBenchmkLatchTrigger_NListeners(b *testing.B, n int) {
	latchPool := make([]Latch, b.N)
	for i := 0; i < b.N; i++ {
		x := New()
		for j := 0; j < n; j++ {
			x.WaitSelectably(make(chan interface{}, 1))
		}
		latchPool[i] = x
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		latchPool[i].Trigger()
	}
}
