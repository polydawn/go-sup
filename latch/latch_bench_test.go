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

		BenchmarkLatchAllocation-4                       1000000               246 ns/op              56 B/op          2 allocs/op
		BenchmarkBaseline_JsonUnmarshalling-4             100000              3106 ns/op             312 B/op          5 allocs/op
		BenchmarkLatchTriggerOnly_0BufDeadend-4          2000000               188 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_1BufDeadend-4          1000000               254 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_2BufDeadend-4          1000000               315 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_4BufDeadend-4          1000000               464 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_8BufDeadend-4           500000               735 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_0UnbufGather-4         2000000               192 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_1UnbufGather-4          500000               611 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_2UnbufGather-4          300000               903 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_4UnbufGather-4          200000              1647 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchTriggerOnly_8UnbufGather-4          100000              2896 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchSubscribe_1BufDeadend-4            1000000               452 ns/op             128 B/op          4 allocs/op
		BenchmarkLatchSubscribe_2BufDeadend-4             300000              1021 ns/op             272 B/op          9 allocs/op
		BenchmarkLatchSubscribe_4BufDeadend-4             200000              2095 ns/op             560 B/op         18 allocs/op
		BenchmarkLatchSubscribe_8BufDeadend-4             100000              4005 ns/op            1136 B/op         35 allocs/op
		BenchmarkLatchFullCycle_0BufDeadend-4            2000000               189 ns/op               0 B/op          0 allocs/op
		BenchmarkLatchFullCycle_1BufDeadend-4             500000               693 ns/op             128 B/op          4 allocs/op
		BenchmarkLatchFullCycle_2BufDeadend-4             200000              1321 ns/op             272 B/op          9 allocs/op
		BenchmarkLatchFullCycle_4BufDeadend-4             100000              2506 ns/op             560 B/op         18 allocs/op
		BenchmarkLatchFullCycle_8BufDeadend-4             100000              4712 ns/op            1136 B/op         35 allocs/op
		BenchmarkFuseTriggerOnly_0Waiters-4              5000000                62.4 ns/op             0 B/op          0 allocs/op
		BenchmarkFuseTriggerOnly_1Waiters-4              1000000               352 ns/op               0 B/op          0 allocs/op
		BenchmarkFuseTriggerOnly_2Waiters-4               500000               645 ns/op               0 B/op          0 allocs/op
		BenchmarkFuseTriggerOnly_4Waiters-4               300000              1373 ns/op               0 B/op          0 allocs/op
		BenchmarkFuseTriggerOnly_8Waiters-4               100000              2747 ns/op               0 B/op          0 allocs/op

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
		  - ~62-68ns per additional buffered gatherer to signal with no blocking receiver; ~200ns baseline -- per BufDeadend tests
		  - ~340-420ns per additional unbuffered gatherer to signal which taps the scheduler; ~200ns basline -- per UnbufGather test
		- Closing an empty/signal channel is O(n) in the blocked reader count!
		  - ~250ns per additional blocked reader
		  - (not shown) Additional reads after the block returns are so cheap they're immeasurable (no suprise there).
		  - (not shown) Changing the fuse chan to buffered size=1 has no impact (no surprise there; it's still blocking-or-not for the reader).
		- Comparing the previous two points:
		  - *Scheduling* is the biggest cost incurred; it's a approx 250ns on these tests.
		    Whether or not there's a blocked reader significantly predominates other factors.
		  - `close` has essentially no other overhead, being a builtin (I presume).
		  - Using lists of channels for the gatherer pattern heaps another 100ns or so onto the minimum scheduling cost.
		  - Remember, comparing these on costs is academic; they fundamentally don't have the same blocking patterns.
		  - Remember, the scale of these costs are ringing in at 1/10th of the cost of a 28-char json unmarshal.
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
	Target: the cost of *triggering*; no one is actually recieving,
	(the message just goes into a buffer -- the scheduler will NOT be tapped).

	Not:
		- allocating the latch
		- allocating the gather chans
		- signing up the gather chans
		- receiving the event (it goes into the chan buffer)
*/
func DoBenchmkLatchTriggerOnly_NBufDeadend(b *testing.B, n int) {
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
func BenchmarkLatchTriggerOnly_0BufDeadend(b *testing.B) { DoBenchmkLatchTriggerOnly_NBufDeadend(b, 0) }
func BenchmarkLatchTriggerOnly_1BufDeadend(b *testing.B) { DoBenchmkLatchTriggerOnly_NBufDeadend(b, 1) }
func BenchmarkLatchTriggerOnly_2BufDeadend(b *testing.B) { DoBenchmkLatchTriggerOnly_NBufDeadend(b, 2) }
func BenchmarkLatchTriggerOnly_4BufDeadend(b *testing.B) { DoBenchmkLatchTriggerOnly_NBufDeadend(b, 4) }
func BenchmarkLatchTriggerOnly_8BufDeadend(b *testing.B) { DoBenchmkLatchTriggerOnly_NBufDeadend(b, 8) }

/*
	Target: the cost of *triggering*, now with someone receiving
	(immediately ready, but unbuffered -- so the scheduler will be tapped).

	Not:
		- allocating the latch
		- allocating the gather chans
		- signing up the gather chans
		- receiving the event -- someone does block for it,
		   but we don't await them.
*/
func DoBenchmkLatchTriggerOnly_NUnbufGather(b *testing.B, n int) {
	subbatch(b, func(b *testing.B) {
		b.StopTimer()
		latchPool := make([]Latch, b.N)
		for i := 0; i < b.N; i++ {
			x := New()
			for j := 0; j < n; j++ {
				ch := make(chan interface{}, 1)
				x.WaitSelectably(ch)
				go func() { <-ch }()
			}
			latchPool[i] = x
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			latchPool[i].Trigger()
		}
	})
}
func BenchmarkLatchTriggerOnly_0UnbufGather(b *testing.B) {
	DoBenchmkLatchTriggerOnly_NUnbufGather(b, 0)
}
func BenchmarkLatchTriggerOnly_1UnbufGather(b *testing.B) {
	DoBenchmkLatchTriggerOnly_NUnbufGather(b, 1)
}
func BenchmarkLatchTriggerOnly_2UnbufGather(b *testing.B) {
	DoBenchmkLatchTriggerOnly_NUnbufGather(b, 2)
}
func BenchmarkLatchTriggerOnly_4UnbufGather(b *testing.B) {
	DoBenchmkLatchTriggerOnly_NUnbufGather(b, 4)
}
func BenchmarkLatchTriggerOnly_8UnbufGather(b *testing.B) {
	DoBenchmkLatchTriggerOnly_NUnbufGather(b, 8)
}

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
func DoBenchmkLatchSubscribe_NBufDeadend(b *testing.B, n int) {
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
func BenchmarkLatchSubscribe_1BufDeadend(b *testing.B) { DoBenchmkLatchSubscribe_NBufDeadend(b, 1) }
func BenchmarkLatchSubscribe_2BufDeadend(b *testing.B) { DoBenchmkLatchSubscribe_NBufDeadend(b, 2) }
func BenchmarkLatchSubscribe_4BufDeadend(b *testing.B) { DoBenchmkLatchSubscribe_NBufDeadend(b, 4) }
func BenchmarkLatchSubscribe_8BufDeadend(b *testing.B) { DoBenchmkLatchSubscribe_NBufDeadend(b, 8) }

/*
	Target: the group cost of allocating chans, subscribing them, and triggering.

	This should be approximately the sum of the subscribe and trigger tests,
	if all in the world adds up nicely.
*/
func DoBenchmkLatchFullCycle_NBufDeadend(b *testing.B, n int) {
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
func BenchmarkLatchFullCycle_0BufDeadend(b *testing.B) { DoBenchmkLatchFullCycle_NBufDeadend(b, 0) }
func BenchmarkLatchFullCycle_1BufDeadend(b *testing.B) { DoBenchmkLatchFullCycle_NBufDeadend(b, 1) }
func BenchmarkLatchFullCycle_2BufDeadend(b *testing.B) { DoBenchmkLatchFullCycle_NBufDeadend(b, 2) }
func BenchmarkLatchFullCycle_4BufDeadend(b *testing.B) { DoBenchmkLatchFullCycle_NBufDeadend(b, 4) }
func BenchmarkLatchFullCycle_8BufDeadend(b *testing.B) { DoBenchmkLatchFullCycle_NBufDeadend(b, 8) }

/*
	Target: the cost of *triggering*.

	We spawn $N goroutines to each block reading on the fuse channel.
	There's matching buffered/no-blocker test because there's no other way to "subscribe".
	This is most comparable to the unbuffered gather chans with blocked readers
	in the the other latch tests.

	Not:
		- allocating the latch
		- blocking for the signals
*/
func DoBenchmkFuseTriggerOnly_NWaiters(b *testing.B, n int) {
	subbatch(b, func(b *testing.B) {
		b.StopTimer()
		fusePool := make([]*fuse, b.N)
		for i := 0; i < b.N; i++ {
			x := NewFuse()
			for j := 0; j < n; j++ {
				go func() { <-x.Selectable() }()
			}
			fusePool[i] = x
		}
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			fusePool[i].Fire()
		}
	})
}
func BenchmarkFuseTriggerOnly_0Waiters(b *testing.B) { DoBenchmkFuseTriggerOnly_NWaiters(b, 0) }
func BenchmarkFuseTriggerOnly_1Waiters(b *testing.B) { DoBenchmkFuseTriggerOnly_NWaiters(b, 1) }
func BenchmarkFuseTriggerOnly_2Waiters(b *testing.B) { DoBenchmkFuseTriggerOnly_NWaiters(b, 2) }
func BenchmarkFuseTriggerOnly_4Waiters(b *testing.B) { DoBenchmkFuseTriggerOnly_NWaiters(b, 4) }
func BenchmarkFuseTriggerOnly_8Waiters(b *testing.B) { DoBenchmkFuseTriggerOnly_NWaiters(b, 8) }
