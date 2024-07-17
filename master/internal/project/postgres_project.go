package project

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/random"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

const (
	// MaxProjectKeyLength is the maximum length of a project key.
	MaxProjectKeyLength = 5
	// MaxProjectKeyPrefixLength is the maximum length of a project key prefix.
	MaxProjectKeyPrefixLength = 3
	// MaxRetries is the maximum number of retries for transaction conflicts.
	MaxRetries = 5
	// ProjectKeyRegex is the regex pattern for a project key.
	ProjectKeyRegex = "^[a-zA-Z0-9]{1,5}$"
)

// getProjectColumns returns a query with the columns for a project, not including experiment
// information.
func getProjectColumns(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		ColumnExpr("p.id").
		ColumnExpr("p.name").
		ColumnExpr("'WORKSPACE_STATE_' || p.state AS state").
		ColumnExpr("p.error_message").
		ColumnExpr("p.workspace_id").
		ColumnExpr("p.description").
		ColumnExpr("(p.archived OR w.archived) as archived").
		ColumnExpr("p.immutable").
		ColumnExpr("p.notes").
		ColumnExpr("(SELECT username FROM users WHERE id = p.user_id) AS username").
		ColumnExpr("p.user_id").
		ColumnExpr("p.key").
		ColumnExpr("w.name as workspace_name").
		ColumnExpr("p.created_at").
		Join("INNER JOIN workspaces w ON w.id = p.workspace_id")
}

// getProjectByIDTx returns a project by its ID using the provided transaction.
func getProjectByIDTx(ctx context.Context, tx bun.Tx, projectID int) (*model.Project, error) {
	p := model.Project{}
	err := tx.NewSelect().
		Model(&p).
		ModelTableExpr("projects as p").
		ColumnExpr(
			"(SELECT MAX(start_time) FROM experiments WHERE project_id = ?) AS last_experiment_started_at",
			projectID,
		).
		ColumnExpr(
			"(SELECT COUNT(*) FROM experiments WHERE project_id = ?) AS num_experiments",
			projectID,
		).
		ColumnExpr(
			"(SELECT COUNT(*) FROM runs WHERE project_id = ?) AS num_runs",
			projectID,
		).
		ColumnExpr(
			"(SELECT COUNT(*) FROM experiments WHERE project_id = ? AND state = 'ACTIVE') AS num_active_experiments",
			projectID,
		).
		Apply(getProjectColumns).
		Where("p.id = ?", projectID).
		Scan(ctx)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProjectByID returns a project by its ID.
func GetProjectByID(ctx context.Context, projectID int) (*model.Project, error) {
	var p *model.Project
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var err error
		p, err = getProjectByIDTx(ctx, tx, projectID)

		return err
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

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

// GetProjectByKey returns a project using its key to identify it.
func GetProjectByKey(ctx context.Context, key string) (*model.Project, error) {
	project := &model.Project{}
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var projectID int
		err := tx.NewSelect().
			Column("id").
			Table("projects").
			Where("key = UPPER(?)", key). // case-insensitive
			Scan(ctx, &projectID)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return db.ErrNotFound
		} else if err != nil {
			return err
		}

		project, err = GetProjectByID(ctx, projectID)
		if err != nil {
			return err
		}
		return nil
	})
	// GetProjectByID handles the case where the project is not found.
	if err != nil {
		return nil, err
	}
	return project, nil
}

// GenerateProjectKey generates a unique project key for a project based on its name.
func generateProjectKey(ctx context.Context, tx bun.Tx, projectName string) (string, error) {
	var key string
	found := true
	for i := 0; i < MaxRetries && found; i++ {
		sanitizedName := strings.ToUpper(regexp.MustCompile("[^a-zA-Z0-9]").ReplaceAllString(projectName, ""))
		prefixLength := min(len(sanitizedName), MaxProjectKeyPrefixLength)
		prefix := sanitizedName[:prefixLength]
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
				var isUsed bool
				err = tx.NewRaw(`
				SELECT EXISTS (
					SELECT 1 FROM local_id_redirect WHERE project_key = ?
				) AS is_used`, p.Key).
					Scan(ctx, &isUsed)
				if err != nil {
					return fmt.Errorf("error creating new project")
				}
				if isUsed {
					return status.Errorf(
						codes.AlreadyExists,
						"error creating new project, provided key '%s' already in use in redirect table",
						p.Key)
				}
				_, err = tx.NewInsert().Model(p).Exec(ctx)
				if err != nil && strings.Contains(err.Error(), db.CodeUniqueViolation) {
					switch errString := err.Error(); {
					case strings.Contains(errString, "projects_key_key"):
						return errors.Wrapf(db.ErrDuplicateRecord, "project key %s is already in use", p.Key)
					case strings.Contains(errString, "projects_name_workspace_id_key"):
						return errors.Wrapf(db.ErrDuplicateRecord, "project name %s is already in use", p.Name)
					default:
						return err
					}
				} else if err != nil {
					return err
				}
				return nil
			},
		)

		switch {
		case err == nil:
			break RetryLoop
		case requestedKey == nil && strings.Contains(err.Error(), fmt.Sprintf("project key %s is already in use", p.Key)):
			log.Debugf("retrying project (%s) insertion due to generated key conflict (%s)", p.Name, p.Key)
			continue // retry
		default:
			break RetryLoop
		}
	}
	return errors.Wrapf(err, "error inserting project %s into database", p.Name)
}

// ValidateProjectKey validates a project key.
func ValidateProjectKey(key string) error {
	switch {
	case len(key) > MaxProjectKeyLength:
		return errors.Errorf("project key cannot be longer than %d characters", MaxProjectKeyLength)
	case len(key) < 1:
		return errors.New("project key cannot be empty")
	case !regexp.MustCompile(ProjectKeyRegex).MatchString(key):
		return errors.Errorf(
			"project key can only contain alphanumeric characters: %s",
			key,
		)
	default:
		return nil
	}
}

// UpdateProject updates a project in the database.
func UpdateProject(
	ctx context.Context,
	projectID int32,
	curUser model.User,
	p *projectv1.PatchProject,
) (*model.Project, error) {
	finalProject := &model.Project{}

	if p.Key != nil {
		// allow user to provide a key, but ensure it is uppercase.
		p.Key.Value = strings.ToUpper(p.Key.Value)
	}

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		currentProject := model.Project{}
		err := tx.NewSelect().Model(&currentProject).
			ModelTableExpr("projects as p").
			Column("p.id").
			ColumnExpr("(p.archived OR w.archived) as archived").
			Column("p.immutable").
			Column("p.name").
			Column("p.description").
			Column("p.key").
			Column("p.workspace_id").
			Where("p.id = ?", projectID).
			Join("INNER JOIN workspaces w ON w.id = p.workspace_id").
			For("UPDATE").
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return db.ErrNotFound
		} else if err != nil {
			return errors.Wrapf(err, "error fetching project (%d) from database", projectID)
		}
		if err = AuthZProvider.Get().CanGetProject(ctx, curUser, currentProject.Proto()); err != nil {
			return authz.SubIfUnauthorized(err, db.ErrNotFound)
		}
		switch {
		case currentProject.Archived:
			return fmt.Errorf("project (%d) is archived and cannot have attributes updated", projectID)
		case currentProject.Immutable:
			return fmt.Errorf("project (%d) is immutable and cannot have attributes updated", projectID)
		}

		var madeChanges bool
		protoProject := currentProject.Proto()
		if p.Name != nil && p.Name.Value != currentProject.Name {
			if err = AuthZProvider.Get().CanSetProjectName(ctx, curUser, protoProject); err != nil {
				return status.Error(codes.PermissionDenied, err.Error())
			}
			log.Infof(
				`project (%d) name changing from "%s" to "%s"`,
				currentProject.ID,
				currentProject.Name,
				p.Name.Value,
			)
			currentProject.Name = p.Name.Value
			madeChanges = true
		}

		if p.Description != nil && p.Description.Value != currentProject.Description {
			if err = AuthZProvider.Get().CanSetProjectDescription(ctx, curUser, protoProject); err != nil {
				return status.Error(codes.PermissionDenied, err.Error())
			}
			log.Infof(
				`project (%d) description changing from "%s" to "%s"`,
				currentProject.ID,
				currentProject.Description,
				p.Description.Value,
			)
			currentProject.Description = p.Description.Value
			madeChanges = true
		}

		if p.Key != nil && p.Key.Value != currentProject.Key {
			if err = AuthZProvider.Get().CanSetProjectKey(ctx, curUser, protoProject); err != nil {
				return status.Error(codes.PermissionDenied, err.Error())
			}
			log.Infof(
				`project (%d) key changing from "%s" to "%s"`,
				currentProject.ID,
				currentProject.Key,
				p.Key.Value,
			)
			currentProject.Key = p.Key.Value
			madeChanges = true
		}

		if !madeChanges {
			finalProject = &currentProject
			return nil
		}

		var isUsed bool
		err = tx.NewRaw(`
		SELECT EXISTS (
			SELECT 1 FROM local_id_redirect WHERE project_key = ? AND project_id != ?
		) AS is_used`, currentProject.Key, currentProject.ID).
			Scan(ctx, &isUsed)
		if err != nil {
			return fmt.Errorf("error updating project %s", currentProject.Name)
		}
		if isUsed {
			return status.Errorf(
				codes.AlreadyExists,
				"error updating project %s, provided key '%s' already in use in redirect table",
				currentProject.Name, currentProject.Key)
		}
		_, err = tx.NewUpdate().Table("projects").
			Set("name = ?", currentProject.Name).
			Set("description = ?", currentProject.Description).
			Set("key = ?", currentProject.Key).
			Where("id = ?", currentProject.ID).
			Exec(ctx)
		if err != nil {
			if strings.Contains(err.Error(), db.CodeUniqueViolation) {
				switch {
				case strings.Contains(err.Error(), "projects_name_key"):
					return status.Errorf(
						codes.AlreadyExists,
						"project name %s is already in use",
						currentProject.Name,
					)
				case strings.Contains(err.Error(), "projects_key_key"):
					return status.Errorf(
						codes.AlreadyExists,
						"project key %s is already in use",
						currentProject.Key,
					)
				}
			}
			return errors.Wrapf(db.MatchSentinelError(err), "error updating project %s", currentProject.Name)
		}

		if _, err = tx.NewRaw(`
		INSERT INTO local_id_redirect (run_id, project_id, project_key, local_id)
		SELECT
			id, project_id, ? as project_key, local_id 
		FROM
			runs
		WHERE project_id = ?
		`, currentProject.Key, currentProject.ID).Exec(ctx); err != nil {
			return fmt.Errorf("adding local id redirect: %w for project %s", err, currentProject.Name)
		}

		// no need to lock the project since it's already been locked above
		err = tx.NewSelect().Model(finalProject).
			ModelTableExpr("projects as p").
			ColumnExpr(
				"(SELECT MAX(start_time) FROM experiments WHERE project_id = ?) AS last_experiment_started_at",
				projectID,
			).
			ColumnExpr(
				"(SELECT COUNT(*) FROM experiments WHERE project_id = ?) AS num_experiments",
				projectID,
			).
			ColumnExpr(
				"(SELECT COUNT(*) FROM experiments WHERE project_id = ? AND state = 'ACTIVE') AS num_active_experiments",
				projectID,
			).
			Apply(getProjectColumns).
			Where("p.id = ?", projectID).
			Scan(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return finalProject, nil
}
