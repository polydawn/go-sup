package latch

import (
	"encoding/json"
	"testing"
)

/*
	RESULTS

		BenchmarkLatchAllocation                 5000000               267 ns/op
		BenchmarkBaseline_JsonUnmarshalling      1000000              2226 ns/op
		BenchmarkLatchTrigger_0Listeners        10000000               187 ns/op
		BenchmarkLatchTrigger_1Listeners         5000000               257 ns/op
		BenchmarkLatchTrigger_2Listeners         5000000               320 ns/op
		BenchmarkLatchTrigger_4Listeners         3000000               462 ns/op
		BenchmarkLatchTrigger_8Listeners         2000000               728 ns/op

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

		Here, we had to go a step further, because of two competing influences:
		the test itself is short, so go bench will run our `b.N` sky high;
		and yet our memory usage will get ridiculous at that `N` and start
		to have other difficult-to-constrain deleterious effects.
		This shouldn't be a common problem; it's most likely a sign of a
		badly targetted benchmark (and this is; it's illustrative only).
		(See git history for an exploration of how memory pressure had a
		*crushing* effect on a *subsequent* benchmark function!  This is a
		situation to avoid at all costs.)
	*/
	subbatch(b, func(b *testing.B) {
		b.StopTimer()
		latchPool := make([]Latch, b.N)
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			latchPool[i] = New()
		}
	})
}

// Totally unrelated.  Just to put things in context.
func BenchmarkBaseline_JsonUnmarshalling(b *testing.B) {
	subbatch(b, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var x interface{}
			json.Unmarshal([]byte(`{"jsonmsg":{"baseline":42}}`), x)
		}
	})
}

/*
	Target: the cost of *triggering*.

	Not:
		- allocating the latch
		- allocating the gather chans
		- signing up the gather chans
*/
func DoBenchmkLatchTrigger_NListeners(b *testing.B, n int) {
	subbatch(b, func(b *testing.B) {
		b.StopTimer()
		latchPool := make([]Latch, b.N)
		for i := 0; i < b.N; i++ {
			x := New()
			for j := 0; j < n; j++ {
				x.WaitSelectably(make(chan interface{}, 1))
			}
			latchPool[i] = x
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			latchPool[i].Trigger()
		}
	})
}
func BenchmarkLatchTrigger_0Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 0) }
func BenchmarkLatchTrigger_1Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 1) }
func BenchmarkLatchTrigger_2Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 2) }
func BenchmarkLatchTrigger_4Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 4) }
func BenchmarkLatchTrigger_8Listeners(b *testing.B) { DoBenchmkLatchTrigger_NListeners(b, 8) }
