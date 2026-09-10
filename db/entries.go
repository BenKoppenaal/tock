package db

import (
	"database/sql"
	"time"
	"tock/model"
)

func (d *DB) StartEntry(taskID int64) (model.Entry, error) {
	now := time.Now()
	res, err := d.conn.Exec(
		`INSERT INTO entries (task_id, start_time) VALUES (?, ?)`,
		taskID, now.Unix(),
	)
	if err != nil {
		return model.Entry{}, err
	}
	id, _ := res.LastInsertId()
	return model.Entry{ID: id, TaskID: taskID, StartTime: now}, nil
}

func (d *DB) StopEntry(id int64, comment string) error {
	_, err := d.conn.Exec(
		`UPDATE entries SET end_time = ?, comment = ? WHERE id = ?`,
		time.Now().Unix(), comment, id,
	)
	return err
}

func (d *DB) ActiveEntry() (*model.Entry, error) {
	row := d.conn.QueryRow(`
		SELECT e.id, e.task_id, t.name, e.start_time, e.comment
		FROM entries e JOIN tasks t ON t.id = e.task_id
		WHERE e.end_time IS NULL
		LIMIT 1
	`)
	var e model.Entry
	var startUnix int64
	err := row.Scan(&e.ID, &e.TaskID, &e.TaskName, &startUnix, &e.Comment)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.StartTime = time.Unix(startUnix, 0)
	return &e, nil
}

func (d *DB) EntriesForDay(day time.Time) ([]model.Entry, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	rows, err := d.conn.Query(`
		SELECT e.id, e.task_id, t.name, e.start_time, e.end_time, e.comment
		FROM entries e JOIN tasks t ON t.id = e.task_id
		WHERE e.start_time >= ? AND e.start_time < ?
		ORDER BY e.start_time
	`, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []model.Entry
	for rows.Next() {
		var e model.Entry
		var startUnix int64
		var endUnix sql.NullInt64
		if err := rows.Scan(&e.ID, &e.TaskID, &e.TaskName, &startUnix, &endUnix, &e.Comment); err != nil {
			return nil, err
		}
		e.StartTime = time.Unix(startUnix, 0)
		if endUnix.Valid {
			t := time.Unix(endUnix.Int64, 0)
			e.EndTime = &t
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
