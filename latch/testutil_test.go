package latch

import "testing"

func subbatch(b *testing.B, fn func(*testing.B)) {
	maxSize := 100 * 1000
	originalN := b.N
	for n := b.N; n > 0; n -= maxSize {
		b.N = n // max left
		if b.N > maxSize {
			b.N = maxSize
		}
		fn(b)
	}
	b.N = originalN
}
