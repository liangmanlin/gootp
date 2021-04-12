package kernel

import "time"

const (
	daySec  int = 24 * hourSec
	hourSec int = 60 * minSec
	minSec  int = 60
)

// unix时间戳
func Now() int64 {
	return time.Now().Unix()
}

// 毫秒unix时间戳
func Now2() int64 {
	return time.Now().UnixNano() / 1e6
}

// 0点的时间戳
func Midnight() int64 {
	t := time.Now()
	now := t.Unix()
	return now - int64(t.Hour()*hourSec+t.Minute()*minSec+t.Second())
}

func WeekDay() int {
	return weekDay(time.Now())
}

func WeekDayFromUnix(now int64) int {
	return weekDay(time.Unix(now, 0))
}

func weekDay(t time.Time) int {
	week := time.Now().Weekday()
	if week == time.Sunday {
		return 7
	}
	return int(week)
}

// 本周1的0点时间戳
func WeekOneMidnight() int64 {
	t := time.Now()
	now := t.Unix()
	week := t.Weekday()
	var dc int
	if week == time.Sunday {
		dc = 6
	} else {
		dc = int(week - time.Monday)
	}
	return now - int64(t.Hour()*hourSec+t.Minute()*minSec+t.Second()+dc*daySec)
}

// 根据时间戳，获取距离1970.1.1 的天数
func DayNumFromUnix(now int64) int {
	return int(now-time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local).Unix()) / daySec
}

// 获取当前时间距离目标时间的天数
func DayNum(t time.Time) int {
	return DayNumFromUnix(Now()) - DayNumFromUnix(t.Unix())
}
