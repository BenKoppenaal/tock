package db

import "tock/model"

func (d *DB) CreateTask(name, note string) (model.Task, error) {
	res, err := d.conn.Exec(`INSERT INTO tasks (name, note) VALUES (?, ?)`, name, note)
	if err != nil {
		return model.Task{}, err
	}
	id, _ := res.LastInsertId()
	return model.Task{ID: id, Name: name, Note: note}, nil
}

func (d *DB) ListTasks(search string) ([]model.Task, error) {
	rows, err := d.conn.Query(`
		SELECT t.id, t.name, t.note
		FROM tasks t
		LEFT JOIN (
			SELECT task_id, MAX(start_time) AS last_tracked
			FROM entries
			GROUP BY task_id
		) e ON e.task_id = t.id
		WHERE t.name LIKE ?
		ORDER BY COALESCE(e.last_tracked, 0) DESC, t.name`,
		"%"+search+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.Name, &t.Note); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) UpdateTask(id int64, name, note string) error {
	_, err := d.conn.Exec(`UPDATE tasks SET name = ?, note = ? WHERE id = ?`, name, note, id)
	return err
}

func (d *DB) DeleteTask(id int64) error {
	_, err := d.conn.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}
