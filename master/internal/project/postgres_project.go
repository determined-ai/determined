package project

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/random"
)

const (
	// MaxProjectKeyLength is the maximum length of a project key.
	MaxProjectKeyLength = 5
	// MaxProjectKeyPrefixLength is the maximum length of a project key prefix.
	MaxProjectKeyPrefixLength = 3
	// MaxRetries is the maximum number of retries for transaction conflicts.
	MaxRetries = 5
)

// ProjectByName returns a project's ID if it exists in the given workspace and is not archived.
func ProjectByName(ctx context.Context, workspaceName string, projectName string) (int, error) {
	workspace, err := workspace.WorkspaceByName(ctx, workspaceName)
	if err != nil {
		return 1, err
	}
	if workspace.Archived {
		return 1, fmt.Errorf("workspace is archived and cannot add new experiments")
	}

	var pID int
	var archived bool
	err = db.Bun().NewSelect().
		Table("projects").
		Column("id").
		Column("archived").
		Where("workspace_id = ?", workspace.ID).
		Where("name = ?", projectName).
		Scan(ctx, &pID, &archived)
	if err == sql.ErrNoRows {
		return 1, db.ErrNotFound
	}
	if err != nil {
		return 1, err
	}
	if archived {
		return 1, fmt.Errorf("project is archived and cannot add new experiments")
	}
	return pID, nil
}

// ProjectIDByName returns a project's ID if it exists in the given workspace.
func ProjectIDByName(ctx context.Context, workspaceID int, projectName string) (*int, error) {
	var pID int
	err := db.Bun().NewRaw("SELECT id FROM projects WHERE name = ? AND workspace_id = ?",
		projectName, workspaceID).Scan(ctx, &pID)
	if err != nil {
		return nil, err
	}
	return &pID, nil
}

// GenerateProjectKey generates a unique project key for a project based on its name.
func generateProjectKey(ctx context.Context, tx bun.Tx, projectName string) (string, error) {
	var key string
	found := true
	for i := 0; i < MaxRetries && found; i++ {
		prefixLength := min(len(projectName), MaxProjectKeyPrefixLength)
		prefix := projectName[:prefixLength]
		suffix := random.String(MaxProjectKeyLength - prefixLength)
		key = strings.ToUpper(prefix + suffix)
		err := tx.NewSelect().Model(&model.Project{}).Where("key = ?", key).For("UPDATE").Scan(ctx)
		found = err == nil
	}
	if found {
		return "", fmt.Errorf("could not generate a unique project key")
	}
	return key, nil
}

// InsertProject inserts a new project into the database.
func InsertProject(
	ctx context.Context,
	p *model.Project,
	requestedKey *string,
) (err error) {
RetryLoop:
	for i := 0; i < MaxRetries; i++ {
		err = db.Bun().RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead},
			func(ctx context.Context, tx bun.Tx) error {
				var err error
				if requestedKey == nil {
					p.Key, err = generateProjectKey(ctx, tx, p.Name)
					if err != nil {
						return err
					}
				} else {
					p.Key = *requestedKey
				}
				_, err = tx.NewInsert().Model(p).Exec(ctx)
				if err != nil {
					return err
				}
				return nil
			},
		)

		switch {
		case err == nil:
			break RetryLoop
		case requestedKey == nil && strings.Contains(err.Error(), "duplicate key value violates unique constraint"):
			log.Debugf("retrying project (%s) insertion due to generated key conflict (%s)", p.Name, p.Key)
			continue // retry
		default:
			break RetryLoop
		}
	}
	return errors.Wrapf(err, "error inserting project %s into database", p.Name)
}
