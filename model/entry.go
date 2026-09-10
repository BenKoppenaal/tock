package model

import "time"

type Entry struct {
	ID        int64
	TaskID    int64
	TaskName  string
	StartTime time.Time
	EndTime   *time.Time
	Comment   string
}

func (e Entry) Duration() time.Duration {
	if e.EndTime == nil {
		return time.Since(e.StartTime)
	}
	return e.EndTime.Sub(e.StartTime)
}

func (e Entry) Active() bool {
	return e.EndTime == nil
}
