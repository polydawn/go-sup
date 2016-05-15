package latch

import (
	"fmt"
	"runtime"
	"testing"
)

func subbatch(b *testing.B, fn func(*testing.B)) {
	maxSize := 100 * 1000
	//maxSize *= 10000 // just.. noop yourself
	originalN := b.N
	fmt.Printf("\norig: %d\n", originalN)
	sanity := 0
	for n := b.N; n > 0; n -= maxSize {
		b.N = n // max left
		if b.N > maxSize {
			b.N = maxSize
		}
		fmt.Printf("  %d left, doing %d\n", n, b.N)
		fn(b)
		b.StopTimer()
		mem := runtime.MemStats{}
		runtime.ReadMemStats(&mem)
		fmt.Printf("    mem time: %d ns\n", mem.PauseTotalNs)
		b.StartTimer()
		sanity += b.N
	}
	b.N = originalN
	fmt.Printf("   done: %d/%d\n", sanity, b.N)
}
