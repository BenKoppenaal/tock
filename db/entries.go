package db

import (
	"database/sql"
	"time"
	"tock/model"
)

func (d *DB) StartEntry(taskID int64, startTime time.Time, comment string) (model.Entry, error) {
	res, err := d.conn.Exec(
		`INSERT INTO entries (task_id, start_time, comment) VALUES (?, ?, ?)`,
		taskID, startTime.Unix(), comment,
	)
	if err != nil {
		return model.Entry{}, err
	}
	id, _ := res.LastInsertId()
	return model.Entry{ID: id, TaskID: taskID, StartTime: startTime, Comment: comment}, nil
}

func (d *DB) StopEntry(id int64, startTime time.Time, endTime time.Time, comment string) error {
	_, err := d.conn.Exec(
		`UPDATE entries SET start_time = ?, end_time = ?, comment = ? WHERE id = ?`,
		startTime.Unix(), endTime.Unix(), comment, id,
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

func (d *DB) UpdateEntry(id int64, startTime time.Time, endTime time.Time, comment string) error {
	_, err := d.conn.Exec(
		`UPDATE entries SET start_time = ?, end_time = ?, comment = ? WHERE id = ?`,
		startTime.Unix(), endTime.Unix(), comment, id,
	)
	return err
}

func (d *DB) TotalTimeByTask() (map[int64]time.Duration, error) {
	rows, err := d.conn.Query(`
		SELECT task_id, SUM(COALESCE(end_time, CAST(strftime('%s','now') AS INTEGER)) - start_time)
		FROM entries
		GROUP BY task_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	totals := make(map[int64]time.Duration)
	for rows.Next() {
		var taskID, seconds int64
		if err := rows.Scan(&taskID, &seconds); err != nil {
			return nil, err
		}
		totals[taskID] = time.Duration(seconds) * time.Second
	}
	return totals, rows.Err()
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
