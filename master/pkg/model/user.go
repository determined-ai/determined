package model

import (
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/guregu/null.v3"
)

var (
	// EmptyPassword is the empty password (i.e., the empty string).
	EmptyPassword = null.NewString("", false)

	// NoPasswordLogin is a password that prevents the user from logging in
	// directly. They can still login via external authentication methods like
	// OAuth.
	NoPasswordLogin = null.NewString("", true)
)

// BCryptCost is a stopgap until we implement sane master-configuration.
const BCryptCost = 15

// UserID is the type for user IDs.
type UserID int

// SessionID is the type for user session IDs.
type SessionID int

// User corresponds to a row in the "users" DB table.
type User struct {
	ID           UserID      `db:"id" json:"id"`
	Username     string      `db:"username" json:"username"`
	PasswordHash null.String `db:"password_hash" json:"-"`
	Admin        bool        `db:"admin" json:"admin"`
	Active       bool        `db:"active" json:"active"`
}

// UserSession corresponds to a row in the "user_sessions" DB table.
type UserSession struct {
	ID     SessionID `db:"id" json:"id"`
	UserID UserID    `db:"user_id" json:"user_id"`
	Expiry time.Time `db:"expiry" json:"expiry"`
}

// A FullUser is a User joined with any other user relations.
type FullUser struct {
	ID       UserID `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Admin    bool   `db:"admin" json:"admin"`
	Active   bool   `db:"active" json:"active"`

	AgentUID   null.Int    `db:"agent_uid" json:"agent_uid"`
	AgentGID   null.Int    `db:"agent_gid" json:"agent_gid"`
	AgentUser  null.String `db:"agent_user" json:"agent_user"`
	AgentGroup null.String `db:"agent_group" json:"agent_group"`
}

// ValidatePassword checks that the supplied password is correct.
func (user User) ValidatePassword(password string) bool {
	// If an empty password was posted, we need to check that the
	// user is a password-less user.
	if password == "" {
		return !user.PasswordHash.Valid
	}

	// If the model's password is empty, then
	// supplied password must be incorrect
	if !user.PasswordHash.Valid {
		return false
	}
	err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash.ValueOrZero()),
		[]byte(password))

	return err == nil
}

// PasswordCanBeModifiedBy checks whether "other" can change the password of "user".
func (user User) PasswordCanBeModifiedBy(other User) bool {
	if other.Username == "determined" {
		return false
	}

	if other.Admin {
		return true
	}

	if other.ID == user.ID {
		return true
	}

	return false
}

// CanCreateUser checks whether the calling user
// has the authority to create other users.
func (user User) CanCreateUser() bool {
	return user.Admin
}

// AdminCanBeModifiedBy checks whether "other" can enable or disable the admin status of "user".
func (user User) AdminCanBeModifiedBy(other User) bool {
	return other.Admin
}

// ActiveCanBeModifiedBy checks whether "other" can enable or disable the active status of "user".
func (user User) ActiveCanBeModifiedBy(other User) bool {
	return other.Admin
}

// UpdatePasswordHash updates the model's password hash employing necessary cryptographic
// techniques.
func (user *User) UpdatePasswordHash(password string) error {
	if password == "" {
		user.PasswordHash = EmptyPassword
	} else {
		passwordHash, err := HashPassword(password)
		if err != nil {
			return errors.Wrap(err, "error updating user password")
		}

		user.PasswordHash = null.StringFrom(passwordHash)
	}
	return nil
}

// HashPassword hashes the user's password.
func HashPassword(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		BCryptCost,
	)
	if err != nil {
		return "", err
	}
	return string(passwordHash), nil
}
