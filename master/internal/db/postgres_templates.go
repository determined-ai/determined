package db

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TemplateByName looks up a config template by name in a database.
func (db *PgDB) TemplateByName(name string) (value model.Template, err error) {
	err = db.Query("get_template", &value, name)
	return value, err
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
		return ErrNotFound
	}
	return nil
}
