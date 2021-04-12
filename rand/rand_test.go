package rand

import (
	"fmt"
	"testing"
)

type qv struct {
	Q int32
	V *data
}

type data struct {
	I	int32
}

func TestRand(t *testing.T) {
	l := []*qv{{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
	}
	r := New()
	fmt.Println(r.RandomQSlice(l, 2, true).([]*data)[1])
	fmt.Println(r.RandomQSlice(l, 4, false))
	fmt.Println(r.RandomNum(1, 100, 10))
}

func BenchmarkRand_RandomQSlice_Single(b *testing.B) {
	l := []*qv{{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
	}
	r := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.RandomQSlice(l, 4, false)
	}
}

func BenchmarkRand_RandomQSlice_Repeated(b *testing.B) {
	l := []*qv{{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
		{10, &data{1}}, {20, &data{2}}, {10, &data{3}},{20,&data{4}},
	}
	r := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.RandomQSlice(l, 4, true)
	}
}