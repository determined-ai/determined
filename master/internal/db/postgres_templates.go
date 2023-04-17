package db

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateList returns all of the config templates in the database.
func (db *PgDB) TemplateList() (values []model.Template, err error) {
	err = db.Query("list_templates", &values)
	return values, err
}

// TemplateByName looks up a config template by name in a database.
func (db *PgDB) TemplateByName(name string) (value model.Template, err error) {
	err = db.Query("get_template", &value, name)
	return value, err
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
		return fmt.Errorf("error setting a template '%v': %w", tpl.Name, err)
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
		return fmt.Errorf("error deleting template '%v': %w", name, err1)
	}
	num, err2 := result.RowsAffected()
	if err2 != nil {
		return fmt.Errorf("error deleting template '%v': %w", name, err2)
	}
	if num != 1 {
		return ErrNotFound
	}
	return nil
}
