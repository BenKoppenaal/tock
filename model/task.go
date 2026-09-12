package model

import "time"

type Task struct {
	ID          int64
	Name        string
	Note        string
	LastTracked *time.Time
}
