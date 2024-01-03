package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

// retrofitSCIMUser "upgrades" an existing user to one tracked in the SCIM table. This is a
// temporary measure for SaaS clusters to migrate existing users to SCIM users.
func retrofitSCIMUser(ctx context.Context, suser *model.SCIMUser, userID model.UserID) (*model.SCIMUser, error) {
	suser.UserID = userID
	id, err := addSCIMUserTx(ctx, db.Bun(), suser)
	if err != nil {
		return nil, err
	}

	suser.ID = id

	return suser, err
}

// AddSCIMUser adds a user as well as additional SCIM-specific fields. If
// the user already exists, this function will return an error.
func AddSCIMUser(ctx context.Context, suser *model.SCIMUser) (*model.SCIMUser, error) {
	if err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		userID, err := AddUserTx(ctx, tx, &model.User{
			Username:     suser.Username,
			DisplayName:  suser.DisplayName,
			Active:       true,
			PasswordHash: suser.PasswordHash,
			Remote:       true,
		})
		if err != nil {
			return err
		}

		suser.UserID = userID

		id, err := addSCIMUserTx(ctx, tx, suser)
		if err != nil {
			return err
		}

		suser.ID = id
		return nil
	}); err != nil {
		return nil, fmt.Errorf("adding SCIM user: %w", err)
	}

	return suser, nil
}

func addSCIMUserTx(ctx context.Context, tx bun.IDB, user *model.SCIMUser) (model.UUID, error) {
	id := model.NewUUID()
	s := struct {
		bun.BaseModel `bun:"table:scim.users"`

		Name          model.SCIMName
		ID            model.UUID
		ExternalID    string
		UserID        model.UserID
		Emails        model.SCIMEmails
		RawAttributes map[string]any
	}{
		Name:          user.Name,
		ID:            id,
		ExternalID:    user.ExternalID,
		UserID:        user.UserID,
		Emails:        user.Emails,
		RawAttributes: user.RawAttributes,
	}

	if _, err := tx.NewInsert().Model(&s).Exec(ctx); err != nil {
		return model.UUID{}, errors.WithStack(err)
	}

	return id, nil
}

// SCIMUserList returns at most count SCIM users starting at startIndex
// (1-indexed). If username is set, restrict results to users with the matching
// username.
func SCIMUserList(ctx context.Context, startIndex, count int, username string) (*model.SCIMUsers, error) {
	var users []*model.SCIMUser
	q := db.Bun().NewSelect().TableExpr("users AS u, scim.users AS s").
		ColumnExpr("s.id, u.username, u.display_name, s.external_id, s.name, s.emails, u.active").
		Where("u.id = s.user_id").Order("id")
	if username != "" {
		q = q.Where("u.username = ?", username)
	}
	if err := q.Scan(ctx, &users); err != nil {
		return nil, errors.WithStack(err)
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
func SCIMUserByID(ctx context.Context, tx bun.IDB, id model.UUID) (*model.SCIMUser, error) {
	var suser model.SCIMUser
	if err := tx.NewSelect().TableExpr("users AS u, scim.users AS s").
		ColumnExpr("s.id, u.username, u.display_name, s.external_id, s.name, s.emails, u.active, s.raw_attributes").
		Where("u.id = s.user_id AND s.id = ?", id).Scan(ctx, &suser); errors.Is(err, sql.ErrNoRows) {
		return nil, errors.WithStack(db.ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &suser, nil
}

// scimUserByAttribute returns the SCIM user with the given value for the given attribute.
func scimUserByAttribute(ctx context.Context, name string, value string) (*model.SCIMUser, error) {
	var suser model.SCIMUser
	if err := db.Bun().NewSelect().TableExpr("users u, scim.users s").
		ColumnExpr("s.id, u.username, u.display_name, s.external_id, s.name, s.emails, u.active, s.raw_attributes").
		Where("u.id = s.user_id AND s.raw_attributes->>? = ?", name, value).
		Scan(ctx, &suser); errors.Is(err, sql.ErrNoRows) {
		return nil, errors.WithStack(db.ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &suser, nil
}

// UserBySCIMAttribute returns the user with the given value for the given SCIM attribute.
func UserBySCIMAttribute(ctx context.Context, name string, value string) (*model.User, error) {
	var user model.User
	if err := db.Bun().NewSelect().TableExpr("users AS u, scim.users AS s").
		ColumnExpr("u.id, u.username, u.display_name, u.active, u.password_hash, u.remote").
		Where("u.id = s.user_id AND s.raw_attributes->>?=?", name, value).
		Scan(ctx, &user); errors.Is(err, sql.ErrNoRows) {
		return nil, errors.WithStack(db.ErrNotFound)
	} else if err != nil {
		return nil, errors.WithStack(err)
	}

	return &user, nil
}

// SetSCIMUser updates fields on an existing SCIM user.
func SetSCIMUser(ctx context.Context, id string, user *model.SCIMUser) (*model.SCIMUser, error) {
	return UpdateUserAndDeleteSession(ctx, id, user,
		[]string{
			"active",
			"emails",
			"external_id",
			"name",
			"username",
			"password_hash",
			"raw_attributes",
			"display_name",
		})
}

// UpdateUserAndDeleteSession updates some fields on an existing SCIM user and deletes the user session if inactive.
func UpdateUserAndDeleteSession(
	ctx context.Context,
	id string,
	user *model.SCIMUser,
	fields []string,
) (*model.SCIMUser, error) {
	if userID := user.ID.String(); id != userID {
		return nil, errors.Errorf("user ID %s does not match updated user ID %s", id, userID)
	}

	var updated *model.SCIMUser
	if err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := updateSCIMUser(ctx, tx, user, set.FromSlice(fields)); err != nil {
			return err
		}

		u, err := SCIMUserByID(ctx, tx, user.ID)
		if err != nil {
			return err
		}

		updated = u

		if !updated.Active {
			subq := tx.NewSelect().Column("u.id").TableExpr("users AS u").
				Join("JOIN scim.users su on u.id = su.user_id").Where("su.id = ?", user.ID)
			if _, err := db.Bun().NewDelete().Table("user_sessions").Where("user_id IN (?)", subq).Exec(ctx); err != nil {
				return fmt.Errorf("deleting user session: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("updating SCIM user & deleting user session if inactive: %w", err)
	}

	return updated, nil
}

func updateSCIMUser(ctx context.Context, tx bun.IDB, user *model.SCIMUser, fieldSet set.Set[string]) error {
	userValues := map[string]interface{}{}
	if fieldSet.Contains("active") {
		userValues["active"] = user.Active
		fieldSet.Remove("active")
	}

	if fieldSet.Contains("username") {
		userValues["username"] = user.Username
		fieldSet.Remove("username")
	}

	if fieldSet.Contains("password_hash") {
		userValues["password_hash"] = user.PasswordHash
		fieldSet.Remove("password_hash")
	}

	if fieldSet.Contains("display_name") {
		userValues["display_name"] = user.DisplayName
		fieldSet.Remove("display_name")
	}

	if len(userValues) > 0 {
		q := tx.NewUpdate().Table("users").Model(&userValues).Where("id = (?)",
			tx.NewSelect().Column("user_id").TableExpr("scim.users AS s").Where("s.id = ?", user.ID))
		if err := execUpdateSCIMUser(ctx, tx, q); err != nil {
			return err
		}
	}

	if len(fieldSet) > 0 {
		q := tx.NewUpdate().ModelTableExpr("?", bun.Safe("scim.users")).
			Column(fieldSet.ToSlice()...).Model(user).Where("id = ?", user.ID)
		if err := execUpdateSCIMUser(ctx, tx, q); err != nil {
			return err
		}
	}

	return nil
}

func execUpdateSCIMUser(ctx context.Context, tx bun.IDB, q *bun.UpdateQuery) error {
	res, err := q.Exec(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	num, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("getting rows affected: %w", err)
	} else if num == 0 {
		return errors.WithStack(db.ErrNotFound)
	}

	return nil
}
