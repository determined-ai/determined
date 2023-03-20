package db

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
)

// scimUserRow is a row in the SCIM table. The SCIM table contains the
// additional information needed to implement the SCIM protocol. This differs
// from model.SCIMUser because the latter is the result of joining a scimUserRow
// with a model.User.
type scimUserRow struct {
	ID            model.UUID             `db:"id"`
	UserID        model.UserID           `db:"user_id"`
	ExternalID    string                 `db:"external_id"`
	Name          model.SCIMName         `db:"name"`
	Emails        model.SCIMEmails       `db:"emails"`
	RawAttributes map[string]interface{} `db:"raw_attributes"`
}

// RetrofitSCIMUser "upgrades" an existing user to one tracked in the SCIM table. This is a
// temporary measure for SaaS clusters to migrate existing users to SCIM users.
func (db *PgDB) RetrofitSCIMUser(suser *model.SCIMUser, userID model.UserID) (*model.SCIMUser,
	error,
) {
	row := &scimUserRow{
		ExternalID:    suser.ExternalID,
		Emails:        suser.Emails,
		Name:          suser.Name,
		RawAttributes: suser.RawAttributes,
	}

	tx, err := db.sql.Beginx()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	id, err := addSCIMUser(tx, userID, row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WithStack(err)
	}

	tx = nil

	added := *suser
	added.ID = id

	return &added, nil
}

// AddSCIMUser adds a user as well as additional SCIM-specific fields. If
// the user already exists, this function will return an error.
func (db *PgDB) AddSCIMUser(suser *model.SCIMUser) (*model.SCIMUser, error) {
	row := &scimUserRow{
		ExternalID:    suser.ExternalID,
		Emails:        suser.Emails,
		Name:          suser.Name,
		RawAttributes: suser.RawAttributes,
	}

	user := &model.User{
		Username:     suser.Username,
		Active:       true,
		PasswordHash: suser.PasswordHash,
	}

	tx, err := db.sql.Beginx()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	userID, err := addUser(tx, user)
	if err != nil {
		return nil, err
	}

	id, err := addSCIMUser(tx, userID, row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WithStack(err)
	}

	tx = nil

	added := *suser
	added.ID = id

	return &added, nil
}

func addSCIMUser(tx *sqlx.Tx, userID model.UserID, row *scimUserRow) (model.UUID, error) {
	next := *row
	next.UserID = userID
	next.ID = model.NewUUID()

	stmt, err := tx.PrepareNamed(`
INSERT INTO scim.users
(id, user_id, external_id, name, emails, raw_attributes)
VALUES (:id, :user_id, :external_id, :name, :emails, :raw_attributes)`)
	if err != nil {
		return model.UUID{}, errors.WithStack(err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(next); err != nil {
		return model.UUID{}, errors.WithStack(err)
	}

	return next.ID, nil
}

// SCIMUserList returns at most count SCIM users starting at startIndex
// (1-indexed). If username is set, restrict results to users with the matching
// username.
func (db *PgDB) SCIMUserList(startIndex, count int, username string) (*model.SCIMUsers, error) {
	var rows *sqlx.Rows
	var err error

	if len(username) == 0 {
		rows, err = db.sql.Queryx(`
SELECT
	s.id, u.username, s.external_id, s.name, s.emails, u.active
FROM users u, scim.users s
WHERE u.id = s.user_id
ORDER BY id`)
	} else {
		rows, err = db.sql.Queryx(`
SELECT
	s.id, u.username, s.external_id, s.name, s.emails, u.active
FROM users u, scim.users s
WHERE u.id = s.user_id AND u.username = $1
ORDER BY id`, username)
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer rows.Close()

	var users []*model.SCIMUser
	for rows.Next() {
		var u model.SCIMUser
		if err := rows.StructScan(&u); err != nil {
			return nil, errors.WithStack(err)
		}
		users = append(users, &u)
	}

	offset := startIndex
	if offset > 0 {
		// startIndex is 1-indexed according to the SCIM specification.
		offset--
	}

	total := len(users)
	if offset > total {
		offset = total
	}
	if offset+count > total {
		count = total - offset
	}

	startIndex = offset + 1

	return &model.SCIMUsers{
		TotalResults: total,
		StartIndex:   startIndex,
		Resources:    users[offset : offset+count],
		ItemsPerPage: count,
	}, nil
}

// SCIMUserByID returns the SCIM user with the given ID.
func (db *PgDB) SCIMUserByID(id model.UUID) (*model.SCIMUser, error) {
	tx, err := db.sql.Beginx()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	suser, err := db.scimUserByID(tx, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WithStack(err)
	}

	tx = nil

	return suser, nil
}

func (db *PgDB) scimUserByID(tx *sqlx.Tx, id model.UUID) (*model.SCIMUser, error) {
	var suser model.SCIMUser
	if err := tx.QueryRowx(`
SELECT
	s.id, u.username, s.external_id, s.name, s.emails, u.active
FROM users u, scim.users s
WHERE u.id = s.user_id AND s.id = $1`, id).StructScan(&suser); err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &suser, nil
}

// SCIMUserByAttribute returns the SCIM user with the given value for the given attribute.
func (db *PgDB) SCIMUserByAttribute(name, value string) (*model.SCIMUser, error) {
	var suser model.SCIMUser
	err := db.sql.QueryRowx(`
SELECT
	s.id, u.username, s.external_id, s.name, s.emails, u.active
FROM users u, scim.users s
WHERE u.id = s.user_id AND s.raw_attributes->>$1 = $2`, name, value).StructScan(&suser)
	if err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &suser, nil
}

// UserBySCIMAttribute returns the user with the given value for the given SCIM attribute.
func (db *PgDB) UserBySCIMAttribute(name, value string) (*model.User, error) {
	var user model.User
	err := db.sql.QueryRowx(`
SELECT
	u.*
FROM users u, scim.users s
WHERE u.id = s.user_id AND s.raw_attributes->>$1 = $2`, name, value).StructScan(&user)
	if err == sql.ErrNoRows {
		return nil, errors.WithStack(ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &user, nil
}

// SetSCIMUser updates fields on an existing SCIM user.
func (db *PgDB) SetSCIMUser(id string, user *model.SCIMUser) (*model.SCIMUser, error) {
	return db.UpdateSCIMUser(id, user,
		[]string{
			"active",
			"emails",
			"external_id",
			"name",
			"username",
			"password_hash",
			"raw_attributes",
		})
}

// UpdateSCIMUser updates some fields on an existing SCIM user.
func (db *PgDB) UpdateSCIMUser(
	id string,
	user *model.SCIMUser,
	fields []string,
) (*model.SCIMUser, error) {
	if e, f := id, user.ID.String(); e != f {
		return nil, errors.Errorf("user ID %s does not match updated user ID %s", e, f)
	}

	tx, err := db.sql.Beginx()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer func() {
		if tx == nil {
			return
		}

		if rErr := tx.Rollback(); rErr != nil {
			log.Errorf("error during rollback: %v", rErr)
		}
	}()

	if err = db.updateSCIMUser(tx, user, fields); err != nil {
		return nil, err
	}

	updated, err := db.scimUserByID(tx, user.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WithStack(err)
	}

	tx = nil

	return updated, nil
}

func (db *PgDB) updateSCIMUser(tx *sqlx.Tx, user *model.SCIMUser, fields []string) error {
	fieldSet := make(map[string]bool)
	for _, v := range fields {
		fieldSet[v] = true
	}

	var usersFields []string
	for _, v := range []string{"active", "username", "password_hash"} {
		if fieldSet[v] {
			usersFields = append(usersFields, v)
		}
		delete(fieldSet, v)
	}

	var scimUsersFields []string
	for v := range fieldSet {
		scimUsersFields = append(scimUsersFields, v)
	}
	sort.Strings(scimUsersFields)

	if fs := usersFields; len(fs) > 0 {
		stmt, err := tx.PrepareNamed(fmt.Sprintf(`
UPDATE users
%v
WHERE id = (SELECT user_id FROM scim.users s WHERE s.id = :id)`, setClause(fs)))
		if err == sql.ErrNoRows {
			return errors.WithStack(ErrNotFound)
		} else if err != nil {
			return errors.WithStack(err)
		}

		defer stmt.Close()

		if _, err := stmt.Exec(user); err != nil {
			return errors.WithStack(err)
		}
	}

	if fs := scimUsersFields; len(fs) > 0 {
		stmt, err := tx.PrepareNamed(fmt.Sprintf(`
UPDATE scim.users
%v
WHERE id = :id`, setClause(fs)))
		if err == sql.ErrNoRows {
			return errors.WithStack(ErrNotFound)
		} else if err != nil {
			return errors.WithStack(err)
		}

		defer stmt.Close()

		if _, err := stmt.Exec(user); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// DeleteSessionsForSCIMUser deletes sessions belonging to a given scim user ID.
func (db *PgDB) DeleteSessionsForSCIMUser(user *model.SCIMUser) error {
	_, err := db.sql.Exec(`
DELETE FROM user_sessions
WHERE user_id IN (SELECT u.id 
                FROM users u
                JOIN scim.users su on u.id = su.user_id
                WHERE su.id = $1)`, user.ID)
	return err
}
