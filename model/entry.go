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
	var d time.Duration

	if e.EndTime == nil {
		d = time.Since(e.StartTime)
	} else {
		d = e.EndTime.Sub(e.StartTime)
	}

	if d < 0 {
		return 0
	}

	return d
}

func (e Entry) Active() bool {
	return e.EndTime == nil
}
