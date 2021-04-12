package rand

import "math/rand"

type Rand struct {
	rand *rand.Rand
}

type mv struct {
	min, max int32
}

type qi struct {
	q   int32
	idx int
}
