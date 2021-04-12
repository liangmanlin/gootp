package gutil

import (
	"sort"
	"testing"
	"unsafe"
)

type rangeConfig struct {
	Min, Max int32
	V        int32
}

type k struct {
	I int32
}

func TestFindRangeValue(t *testing.T) {
	l := []rangeConfig{{1, 2, 1}, {3, 4, 2}, {5, 6, 3}, {7, 199, 4}}
	t.Logf("result:%#v", FindRangeValue(l, 100))
}

func BenchmarkFindRangeValue(b *testing.B) {
	l := []rangeConfig{{1, 2, 1}, {3, 4, 2}, {5, 6, 3}, {7, 199, 4}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindRangeValue(l, 100)
	}
}

func BenchmarkSliceDelInt32(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := []int32{1, 2, 3, 4, 5, 6, 7, 8}
		SliceDelInt32(l, 3)
	}
}

func TestSort(t *testing.T) {
	il := []int32{5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
	}
	Sort(il, func(data unsafe.Pointer, i, j int) bool {
		l := *(*[]int32)(data)
		return l[i] >= l[j]
	}, func(data unsafe.Pointer, i, j int) {
		l := *(*[]int32)(data)
		l[i], l[j] = l[j], l[i]
	})
	t.Logf("%v", il)
}

func BenchmarkSortInt32(b *testing.B) {
	l := []int32{5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
	}
	cmp := func(data unsafe.Pointer, i, j int) bool {
		l := *(*[]int32)(data)
		return l[i] >= l[j]
	}
	swap := func(data unsafe.Pointer, i, j int) {
		l := *(*[]int32)(data)
		l[i], l[j] = l[j], l[i]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sort(l, cmp, swap)
	}
}

func BenchmarkSort(b *testing.B) {
	l := []int{5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
		5, 4, 2, 1, 7, 8, 9, 10, 11, 100, 3,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sort.Ints(l)
	}
}

func BenchmarkCeil(b *testing.B) {
	f := float32(1.1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Ceil(f)
	}
}

func BenchmarkRound(b *testing.B) {
	f := float32(1.1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Round(f)
	}
}

func BenchmarkTrunc(b *testing.B) {
	f := float32(1.1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Trunc(f)
	}
}
