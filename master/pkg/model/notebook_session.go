package model

import "github.com/uptrace/bun"

// NotebookSession corresponds to a row in the "notebook_sessions" DB table.
type NotebookSession struct {
	bun.BaseModel `bun:"table:notebook_sessions"`
	ID            SessionID `db:"id" bun:"id,pk,autoincrement" json:"id"`
	TaskID        TaskID    `db:"task_id" bun:"task_id" json:"task_id"`
	UserSessionID SessionID `db:"user_session_id" bun:"user_session_id" json:"user_session_id"`
	Token         *string   `db:"token" bun:"token" json:"token"`
}
