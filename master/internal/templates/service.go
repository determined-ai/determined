package templates

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateByName looks up a config template by name in a database.
func TemplateByName(ctx context.Context, name string) (model.Template, error) {
	var dest model.Template
	err := db.Bun().NewSelect().Table("templates").
		ColumnExpr("*").
		Where("name = ?", name).
		Scan(ctx, &dest)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return model.Template{}, db.ErrNotFound
	case err != nil:
		return dest, fmt.Errorf("fetching template %s from database: %w", name, err)
	}
	return dest, nil
}

// UnmarshalTemplateConfig unmarshals the template config into `o` and returns api-ready errors.
func UnmarshalTemplateConfig(
	ctx context.Context,
	name string,
	user *model.User,
	out interface{},
	disallowUnknownFields bool,
) error {
	tpl, err := TemplateByName(ctx, name)
	switch {
	case errors.Is(err, db.ErrNotFound):
		return api.NotFoundErrs("template", name, true)
	case err != nil:
		return err
	}

	permErr, err := AuthZProvider.Get().CanViewTemplate(
		ctx,
		user,
		model.AccessScopeID(tpl.WorkspaceID),
	)
	switch {
	case err != nil:
		return err
	case permErr != nil:
		return api.NotFoundErrs("template", name, true)
	}

	var opts []yaml.JSONOpt
	if disallowUnknownFields {
		opts = append(opts, yaml.DisallowUnknownFields)
	}
	err = yaml.Unmarshal(tpl.Config, out, opts...)
	if err != nil {
		return fmt.Errorf("yaml.Unmarshal(template=%s): %w", name, err)
	}
	return nil
}

// DeleteWorkspaceTemplates deletes all the templates in a workspace.
func DeleteWorkspaceTemplates(ctx context.Context, workspaceID int) error {
	_, err := db.Bun().NewDelete().
		Model(&model.Template{}).
		Where("workspace_id = ?", workspaceID).
		Exec(ctx)
	return err
}
