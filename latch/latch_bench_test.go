package latch

import (
	"encoding/json"
	"runtime"
	"testing"
)

func init() {
	runtime.GOMAXPROCS(4)
}

/*
	Note: running with `-benchtime 200ms` (or even 100) may be a perfectly
	valid time-saving choice; more does not appear to significantly
	improve the consistency of results.

	RESULTS

		BenchmarkLatchAllocation-4               1000000               254 ns/op              56 B/op          2 allocs/op
		BenchmarkBaseline_JsonUnmarshalling-4     100000              3188 ns/op             312 B/op          5 allocs/op
		BenchmarkLatchTriggerOnly_0Gatherers-4   2000000               193 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_1Gatherers-4   1000000               256 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_2Gatherers-4   1000000               318 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_4Gatherers-4   1000000               465 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_8Gatherers-4    500000               733 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchSubscribe_1Gatherers-4      500000               591 ns/op             128 B/op          4 allocs/op
		BenchmarkLatchSubscribe_2Gatherers-4      200000              1315 ns/op             272 B/op          9 allocs/op
		BenchmarkLatchSubscribe_4Gatherers-4      100000              2641 ns/op             560 B/op         18 allocs/op
		BenchmarkLatchSubscribe_8Gatherers-4       50000              5216 ns/op            1136 B/op         35 allocs/op
		BenchmarkLatchFullCycle_0Gatherers-4     2000000               193 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchFullCycle_1Gatherers-4      300000               871 ns/op             128 B/op          4 allocs/op
		BenchmarkLatchFullCycle_2Gatherers-4      200000              1724 ns/op             272 B/op          9 allocs/op
		BenchmarkLatchFullCycle_4Gatherers-4      100000              3052 ns/op             560 B/op         18 allocs/op
		BenchmarkLatchFullCycle_8Gatherers-4       50000              6273 ns/op            1136 B/op         35 allocs/op

	Cautions:

		- The `BenchmarkLatchTrigger_*Listeners` family uses unbuffered channels,
		  because we don't want to start measuring the vageries of goroutine scheduling.
		- The json "canary" test is phased by way more weird stuff than you'd like to think.
		  - The amount of GC work created by the first test phases it (yes,
		    regardless of the benchmark framework's attempt to compensate for that).
		  - Bizarrely, maxprocs affects the json canary more than any other test.

	Observations:

		- Setting one listener costs about half (or less) as much as a small json parse.
		- Subscribing gatherer chans to the latch is O(n) (no surprise there).
		  - ~700ns per additional gatherer
		- Triggering the latch is O(n) in the gatherer count (no surprise there).
		  - ~62-68ns per additional gatherer to signal; ~200ns baseline.
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
		- receiving the event (it goes into the chan buffer)
*/
func DoBenchmkLatchTriggerOnly_NGatherers(b *testing.B, n int) {
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
func BenchmarkLatchTriggerOnly_0Gatherers(b *testing.B) { DoBenchmkLatchTriggerOnly_NGatherers(b, 0) }
func BenchmarkLatchTriggerOnly_1Gatherers(b *testing.B) { DoBenchmkLatchTriggerOnly_NGatherers(b, 1) }
func BenchmarkLatchTriggerOnly_2Gatherers(b *testing.B) { DoBenchmkLatchTriggerOnly_NGatherers(b, 2) }
func BenchmarkLatchTriggerOnly_4Gatherers(b *testing.B) { DoBenchmkLatchTriggerOnly_NGatherers(b, 4) }
func BenchmarkLatchTriggerOnly_8Gatherers(b *testing.B) { DoBenchmkLatchTriggerOnly_NGatherers(b, 8) }

/*
	Target: the cost of allocating a new chan and subscribing it.

	Not:
		- allocating the latch
		- triggering the latch
		- receiving the event (it goes into the chan buffer)

	Note: you don't wanna do this one with zero gatherers, because it's
	basically testing a noop but doing so in a way that hammers pause button
	and thus wastes a ton of wall clock time on memory stats that don't matter.

	Note: all these are subscribes before the trigger; none after.
	There is no test for the post-trigger subscribes.
	It's not clear what use this would be, because they essentially hit
	the same lock mechanism (for that matter, most of what we're testing
	here with increasing chan counts is the chan alloc, and then an
	`append` call inside the latch; the lock is also all the same here).
*/
func DoBenchmkLatchSubscribe_NGatherers(b *testing.B, n int) {
	subbatch(b, func(b *testing.B) {
		b.StopTimer()
		latchPool := make([]Latch, b.N)
		for i := 0; i < b.N; i++ {
			latchPool[i] = New()
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			x := latchPool[i]
			for j := 0; j < n; j++ {
				x.WaitSelectably(make(chan interface{}, 1))
			}
		}
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			latchPool[i].Trigger()
		}
		b.StartTimer()
	})
}
func BenchmarkLatchSubscribe_1Gatherers(b *testing.B) { DoBenchmkLatchSubscribe_NGatherers(b, 1) }
func BenchmarkLatchSubscribe_2Gatherers(b *testing.B) { DoBenchmkLatchSubscribe_NGatherers(b, 2) }
func BenchmarkLatchSubscribe_4Gatherers(b *testing.B) { DoBenchmkLatchSubscribe_NGatherers(b, 4) }
func BenchmarkLatchSubscribe_8Gatherers(b *testing.B) { DoBenchmkLatchSubscribe_NGatherers(b, 8) }

/*
	Target: the group cost of allocating chans, subscribing them, and triggering.

	This should be approximately the sum of the subscribe and trigger tests,
	if all in the world adds up nicely.
*/
func DoBenchmkLatchFullCycle_NGatherers(b *testing.B, n int) {
	subbatch(b, func(b *testing.B) {
		b.StopTimer()
		latchPool := make([]Latch, b.N)
		for i := 0; i < b.N; i++ {
			latchPool[i] = New()
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			x := latchPool[i]
			for j := 0; j < n; j++ {
				x.WaitSelectably(make(chan interface{}, 1))
			}
			x.Trigger()
		}
	})
}
func BenchmarkLatchFullCycle_0Gatherers(b *testing.B) { DoBenchmkLatchFullCycle_NGatherers(b, 0) }
func BenchmarkLatchFullCycle_1Gatherers(b *testing.B) { DoBenchmkLatchFullCycle_NGatherers(b, 1) }
func BenchmarkLatchFullCycle_2Gatherers(b *testing.B) { DoBenchmkLatchFullCycle_NGatherers(b, 2) }
func BenchmarkLatchFullCycle_4Gatherers(b *testing.B) { DoBenchmkLatchFullCycle_NGatherers(b, 4) }
func BenchmarkLatchFullCycle_8Gatherers(b *testing.B) { DoBenchmkLatchFullCycle_NGatherers(b, 8) }
