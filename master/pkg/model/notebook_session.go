package model

import "github.com/uptrace/bun"

// NotebookSession corresponds to a row in the "notebook_sessions" DB table.
type NotebookSession struct {
	bun.BaseModel `bun:"table:notebook_sessions"`
	ID            SessionID `db:"id" bun:"id,pk,autoincrement" json:"id"`
	TaskID        TaskID    `db:"task_id" bun:"task_id" json:"task_id"`
	// SessionID is only used for notebooks launched before UserID column was added.
	SessionID *SessionID `bun:"-" json:"user_session_id"`
	UserID    UserID     `db:"user_id" bun:"user_id" json:"user_id"`
}

// NotebookSessionEnvVar is the environment variable name for notebook task tokens.
const NotebookSessionEnvVar = "DET_NOTEBOOK_TOKEN"
