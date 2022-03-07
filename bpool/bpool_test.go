package bpool

import "testing"

func BenchmarkNewAndFree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := New(128)
		buf.Free()
	}
}
