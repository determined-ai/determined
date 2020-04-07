package db

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateList returns all of the config templates in the database.
func (db *PgDB) TemplateList() ([]*model.Template, error) {
	rows, err := db.sql.Queryx(`
SELECT name, config
FROM templates`)
	if err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.Wrap(err, "querying for template list")
	}

	defer rows.Close()

	var tpls []*model.Template
	for rows.Next() {
		var tpl model.Template
		if err = rows.StructScan(&tpl); err != nil {
			return nil, errors.Wrap(err, "reading template row")
		}
		tpls = append(tpls, &tpl)
	}

	return tpls, nil
}

// TemplateByName looks up a config template by name in a database.
func (db *PgDB) TemplateByName(name string) (*model.Template, error) {
	var template model.Template
	if err := db.query(`
SELECT name, config
FROM templates
WHERE name = $1`, &template, name); err != nil {
		return nil, err
	}
	return &template, nil
}

// UpsertTemplate creates or updates a config template.
func (db *PgDB) UpsertTemplate(tpl *model.Template) error {
	if len(tpl.Name) == 0 {
		return errors.New("error setting a template: empty name")
	}
	err := db.namedExecOne(`
INSERT INTO templates (name, config)
VALUES (:name, :config)
ON CONFLICT (name)
DO
UPDATE SET config=:config`, tpl)
	if err != nil {
		return errors.Wrapf(err, "error setting a template '%v'", tpl.Name)
	}
	return nil
}

// DeleteTemplate deletes an existing experiment config template.
func (db *PgDB) DeleteTemplate(name string) error {
	if len(name) == 0 {
		return errors.New("error deleting template: empty name")
	}
	result, err1 := db.sql.Exec(`
DELETE FROM templates
WHERE name=$1`, name)
	if err1 != nil {
		return errors.Wrapf(err1, "error deleting template '%v'", name)
	}
	num, err2 := result.RowsAffected()
	if err2 != nil {
		return errors.Wrapf(err2, "error deleting template '%v'", name)
	}
	if num != 1 {
		return errors.WithStack(ErrNotFound)
	}
	return nil
}
