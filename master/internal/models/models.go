package models

import (
	"context"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func ByID(ctx context.Context, id int) (model.Model, error) {
	var out model.Model
	err := db.Bun().NewSelect().
		Model(&out).
		// XXX: a better way to do a count query as a subquery?
		ColumnExpr(`(select count (*) from model_versions
					where model_versions.model_id = models.id) as num_versions`).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return model.Model{}, errors.Wrapf(err, "error selecting model(%v)", id)
	}
	return out, err
}

func ByName(ctx context.Context, name string) (model.Model, error) {
	var out model.Model
	err := db.Bun().NewSelect().
		Model(&out).
		// XXX: a better way to do a count query as a subquery?
		ColumnExpr(`(select count (*) from model_versions
					where model_versions.model_id = models.id) as num_versions`).
		Where("name = ?", name).
		Scan(ctx)
	if err != nil {
		return model.Model{}, errors.Wrapf(err, "error selecting model(%v)", name)
	}
	return out, err
}

func VersionByID(ctx context.Context, id int) (model.ModelVersion, error) {
	var out model.ModelVersion
	err := db.Bun().NewSelect().
		Model(&out).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return model.ModelVersion{},
			errors.Wrapf(err, "error selecting model(id=%v)", id)
	}
	return out, nil
}
