package timer

type TimerKey struct {
	Key int32
	ID  int32
}

type Timer struct {
	m map[TimerKey]*timer
	tmp map[TimerKey]*timer
	isLooping bool
}

type timer struct {
	time  int64
	times int32
	inv int32
	f   interface{}
	arg []interface{}
}
