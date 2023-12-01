package user

import (
	"context"
	"database/sql"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
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

// retrofitSCIMUser "upgrades" an existing user to one tracked in the SCIM table. This is a
// temporary measure for SaaS clusters to migrate existing users to SCIM users.
func retrofitSCIMUser(ctx context.Context, suser *model.SCIMUser, userID model.UserID) (*model.SCIMUser, error) {
	added := *suser
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		id, err := addSCIMUserTx(ctx, tx, userID, &scimUserRow{
			ExternalID:    suser.ExternalID,
			Emails:        suser.Emails,
			Name:          suser.Name,
			RawAttributes: suser.RawAttributes,
		})
		if err != nil {
			return err
		}
		added.ID = id
		return nil
	})

	return &added, err
}

// AddSCIMUser adds a user as well as additional SCIM-specific fields. If
// the user already exists, this function will return an error.
func AddSCIMUser(ctx context.Context, suser *model.SCIMUser) (*model.SCIMUser, error) {
	added := *suser
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		userID, err := Add(
			ctx, &model.User{
				Username:     suser.Username,
				Active:       true,
				PasswordHash: suser.PasswordHash,
				Remote:       true,
			}, nil)
		if err != nil {
			return err
		}

		id, err := addSCIMUserTx(
			ctx, tx, userID, &scimUserRow{
				ExternalID:    suser.ExternalID,
				Emails:        suser.Emails,
				Name:          suser.Name,
				RawAttributes: suser.RawAttributes,
			})
		if err != nil {
			return err
		}

		added.ID = id
		return nil
	})

	return &added, err
}

func addSCIMUserTx(ctx context.Context, tx bun.IDB, userID model.UserID, row *scimUserRow) (model.UUID, error) {
	next := *row
	next.UserID = userID
	next.ID = model.NewUUID()

	if _, err := tx.NewInsert().Table("scim.users").Model(next).Exec(ctx); err != nil {
		return model.UUID{}, errors.WithStack(err)
	}

	return next.ID, nil
}

// SCIMUserList returns at most count SCIM users starting at startIndex
// (1-indexed). If username is set, restrict results to users with the matching
// username.
func SCIMUserList(ctx context.Context, startIndex, count int, username string) (*model.SCIMUsers, error) {
	var rows *sqlx.Rows
	var err error
	if len(username) == 0 {
		err = db.Bun().NewSelect().TableExpr("users AS u, scim.users AS s").
			ColumnExpr("s.id, u.username, s.external_id, s.name, e.emails, u.inactive").
			Where("u.id = s.user_id").Order("id").Scan(ctx, rows)
	} else {
		err = db.Bun().NewSelect().TableExpr("users AS u, scim.users AS s").
			ColumnExpr("s.id, u.username, s.external_id, s.name, e.emails, u.inactive").
			Where("u.id = s.user_id AND u.username = ?", username).Order("id").Scan(ctx, rows)
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

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
func SCIMUserByID(ctx context.Context, id model.UUID) (*model.SCIMUser, error) {
	var suser *model.SCIMUser
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		s, err := scimUserByID(ctx, tx, id)
		if err != nil {
			return err
		}
		suser = s
		return nil
	})

	return suser, err
}

func scimUserByID(ctx context.Context, tx bun.IDB, id model.UUID) (*model.SCIMUser, error) {
	var suser model.SCIMUser
	if err := tx.NewSelect().TableExpr("users AS u, scim.users AS s").
		ColumnExpr("s.id, u.username, s.external_id, s.name, s.emails, u.active").
		Where("u.id = s.user_id AND s.id = ?", id).Scan(ctx, &suser); err == sql.ErrNoRows {
		return nil, errors.WithStack(db.ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &suser, nil
}

// scimUserByAttribute returns the SCIM user with the given value for the given attribute.
func scimUserByAttribute(ctx context.Context, name string, value string) (*model.SCIMUser, error) {
	var suser model.SCIMUser
	err := db.Bun().NewSelect().TableExpr("users u, scim.users s").
		ColumnExpr("s.id, u.username, s.external_id, s.name, s.emails, u.active").
		Where("u.id = s.user_id AND s.raw_attributes->>? = ?", name, value).Scan(ctx, &suser)
	if err == sql.ErrNoRows {
		return nil, errors.WithStack(db.ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &suser, nil
}

// UserBySCIMAttribute returns the user with the given value for the given SCIM attribute.
func UserBySCIMAttribute(ctx context.Context, name string, value string) (*model.User, error) {
	var user model.User
	err := db.Bun().NewSelect().TableExpr("users AS u, scim.users AS s").
		Where("u.id = s.user_id AND s.raw_attributes->>?=?", name, value).Scan(ctx, &user)

	if err == sql.ErrNoRows {
		return nil, errors.WithStack(db.ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &user, nil
}

// SetSCIMUser updates fields on an existing SCIM user.
func SetSCIMUser(ctx context.Context, id string, user *model.SCIMUser) (*model.SCIMUser, error) {
	return UpdateSCIMUser(ctx, id, user,
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
func UpdateSCIMUser(ctx context.Context, id string, user *model.SCIMUser, fields []string) (*model.SCIMUser, error) {
	if e, f := id, user.ID.String(); e != f {
		return nil, errors.Errorf("user ID %s does not match updated user ID %s", e, f)
	}

	var updated *model.SCIMUser
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := updateSCIMUser(ctx, tx, user, fields); err != nil {
			return err
		}

		u, err := scimUserByID(ctx, tx, user.ID)
		if err != nil {
			return err
		}

		updated = u
		return nil
	})

	return updated, err
}

func updateSCIMUser(ctx context.Context, tx bun.IDB, user *model.SCIMUser, fields []string) error {
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
		subq := tx.NewSelect().Column("user_id").Table("scim.users AS s").Where("s.id = :id")
		_, err := tx.NewUpdate().Table("users").Where("id = (?)", subq).Exec(ctx)

		if err == sql.ErrNoRows {
			return errors.WithStack(db.ErrNotFound)
		} else if err != nil {
			return errors.WithStack(err)
		}
	}

	if fs := scimUsersFields; len(fs) > 0 {
		_, err := tx.NewUpdate().Table("scim.users").Where("id = :id").Exec(ctx)
		if err == sql.ErrNoRows {
			return errors.WithStack(db.ErrNotFound)
		} else if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// DeleteSessionsForSCIMUser deletes sessions belonging to a given scim user ID.
func DeleteSessionsForSCIMUser(ctx context.Context, user *model.SCIMUser) error {
	subq := db.Bun().NewSelect().Column("u.id").TableExpr("users AS u").
		Join("JOIN scim.users su on u.id = su.user_id").Where("su.id = ?", user.ID)
	_, err := db.Bun().NewDelete().Table("user_sessions").Where("user_id IN (?)", subq).Exec(ctx)
	return err
}
