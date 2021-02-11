package model

// TaskSession corresponds to a row in the "task_sessions" DB table.
type TaskSession struct {
	ID     SessionID `db:"id" json:"id"`
	TaskID string    `db:"task_id" json:"task_id"`
}
