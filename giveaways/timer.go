package giveaways

import (
	"fmt"
	"time"
)

type Timer struct {
	StartTime int64
	Duration  int
	timer     *time.Timer
}

func NewTimer(startTimeUnix int64, durationMinutes int, callback func()) *Timer {
	startTime := time.Unix(startTimeUnix, 0)
	endTime := startTime.Add(time.Duration(durationMinutes) * time.Second)

	now := time.Now()
	timeRemaining := endTime.Sub(now)

	var timer *time.Timer
	if timeRemaining <= 0 {
		timer = time.NewTimer(0)
		timer.Stop()
	} else {
		fmt.Println("what", timeRemaining)
		timer = time.AfterFunc(timeRemaining, callback)
	}

	return &Timer{
		StartTime: startTimeUnix,
		Duration:  durationMinutes,
		timer:     timer,
	}
}

func (t *Timer) Stop() bool {
	return t.timer.Stop()
}

func (t *Timer) Reset(startTimeUnix int64, durationMinutes int) bool {
	startTime := time.Unix(startTimeUnix, 0)
	endTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)

	now := time.Now()
	timeRemaining := endTime.Sub(now)

	t.StartTime = startTimeUnix
	t.Duration = durationMinutes

	if timeRemaining <= 0 {
		return t.timer.Reset(0)
	}
	return t.timer.Reset(timeRemaining)
}

/*
startTime := time.Now().Add(-1 * time.Minute).Unix()

timer := NewTimer(startTime, 3, func() {
    fmt.Println("Timer finished!")
})
*/
