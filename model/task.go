package model

import "time"

type Task struct {
	ID          int64
	Name        string
	Note        string
	Archived    bool
	LastTracked *time.Time
}
