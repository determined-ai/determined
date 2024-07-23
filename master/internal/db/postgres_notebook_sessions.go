package db

import (
	"context"
	"fmt"

	"github.com/o1egl/paseto"

	"github.com/determined-ai/determined/master/pkg/model"
)

// StartNotebookSession persists a new notebook session row into the database.
func StartNotebookSession(
	ctx context.Context,
	userID model.UserID,
	taskID model.TaskID,
) error {
	notebookSession := &model.NotebookSession{
		UserID: userID,
		TaskID: taskID,
	}

	if _, err := Bun().NewInsert().Model(notebookSession).
		Returning("id").Exec(ctx, &notebookSession.ID); err != nil {
		return fmt.Errorf("failed to create notebook session for task (%s): %w", taskID, err)
	}

	return nil
}

// GenerateNotebookSessionToken generates a token for a notebook session.
func GenerateNotebookSessionToken(
	userID model.UserID,
	taskID model.TaskID,
) (string, error) {
	notebookSession := &model.NotebookSession{
		UserID: userID,
		TaskID: taskID,
	}

	v2 := paseto.NewV2()
	token, err := v2.Sign(GetTokenKeys().PrivateKey, notebookSession, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate task authentication token: %w", err)
	}
	return token, nil
}

// DeleteNotebookSessionByTask deletes the notebook session associated with the task.
func DeleteNotebookSessionByTask(
	ctx context.Context,
	taskID model.TaskID,
) error {
	if _, err := Bun().NewDelete().
		Table("notebook_sessions").
		Where("task_id = ?", taskID).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete notebook session for task (%s): %w", taskID, err)
	}
	return nil
}
