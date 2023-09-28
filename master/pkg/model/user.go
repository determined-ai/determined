package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/guregu/null.v3"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/proto/pkg/userv1"
)

var (
	// DefaultPassword is a default password set in master config.
	DefaultPassword = null.NewString("", false)

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
	bun.BaseModel `bun:"table:users"`
	ID            UserID      `db:"id" bun:"id,pk,autoincrement" json:"id"`
	Username      string      `db:"username" json:"username"`
	PasswordHash  null.String `db:"password_hash" json:"-"`
	DisplayName   null.String `db:"display_name" json:"display_name"`
	Admin         bool        `db:"admin" json:"admin"`
	Active        bool        `db:"active" json:"active"`
	ModifiedAt    time.Time   `db:"modified_at" json:"modified_at"`
	Remote        bool        `db:"remote" json:"remote"`
	LastLogin     time.Time   `db:"last_login" json:"last_login"`
}

// UserSession corresponds to a row in the "user_sessions" DB table.
type UserSession struct {
	bun.BaseModel `bun:"table:user_sessions"`
	ID            SessionID `db:"id" json:"id"`
	UserID        UserID    `db:"user_id" json:"user_id"`
	Expiry        time.Time `db:"expiry" json:"expiry"`
}

// A FullUser is a User joined with any other user relations.
type FullUser struct {
	ID          UserID      `db:"id" json:"id"`
	DisplayName null.String `db:"display_name" json:"display_name"`
	Username    string      `db:"username" json:"username"`
	Name        string      `db:"name" json:"name"`
	Admin       bool        `db:"admin" json:"admin"`
	Active      bool        `db:"active" json:"active"`
	ModifiedAt  time.Time   `db:"modified_at" json:"modified_at"`
	Remote      bool        `db:"remote" json:"remote"`
	LastLogin   time.Time   `db:"last_login" json:"last_login"`

	AgentUID   null.Int    `db:"agent_uid" json:"agent_uid"`
	AgentGID   null.Int    `db:"agent_gid" json:"agent_gid"`
	AgentUser  null.String `db:"agent_user" json:"agent_user"`
	AgentGroup null.String `db:"agent_group" json:"agent_group"`
}

// ToUser converts a FullUser model to just a User model.
func (u FullUser) ToUser() User {
	return User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: null.String{},
		DisplayName:  u.DisplayName,
		Admin:        u.Admin,
		Active:       u.Active,
		ModifiedAt:   u.ModifiedAt,
		Remote:       u.Remote,
		LastLogin:    u.LastLogin,
	}
}

// SetDefaultPassword initializes default password.
func (user User) SetDefaultPassword(password string) error {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		BCryptCost,
	)
	DefaultPassword = null.StringFrom(string(passwordHash))
	if err != nil {
		return err
	}
	return nil
}

// ValidatePassword checks that the supplied password is correct.
func (user User) ValidatePassword(password string) bool {
	// If the model's password is empty, then
	// check for default
	if !user.PasswordHash.Valid {
		// If default password is not set, then
		// supplied password must be empty
		if !DefaultPassword.Valid {
			return password == ""
		}
		err := bcrypt.CompareHashAndPassword(
			[]byte(DefaultPassword.ValueOrZero()),
			[]byte(password))
		return err == nil
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash.ValueOrZero()),
		[]byte(password))

	return err == nil
}

// UpdatePasswordHash updates the model's password hash employing necessary cryptographic
// techniques.
func (user *User) UpdatePasswordHash(password string) error {
	if password == "" {
		user.PasswordHash = null.NewString("", false)
	} else {
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(password),
			BCryptCost,
		)
		if err != nil {
			return err
		}

		user.PasswordHash = null.StringFrom(string(passwordHash))
	}
	return nil
}

// Proto converts a user to its protobuf representation.
func (user *User) Proto() *userv1.User {
	return &userv1.User{
		Id:          int32(user.ID),
		Username:    user.Username,
		DisplayName: user.DisplayName.ValueOrZero(),
		Admin:       user.Admin,
		Active:      user.Active,
		ModifiedAt:  timestamppb.New(user.ModifiedAt),
		Remote:      user.Remote,
		LastLogin:   timestamppb.New(user.LastLogin),
	}
}

// Users is a slice of User objects—primarily useful for its methods.
type Users []User

// Proto converts a slice of users to its protobuf representation.
func (users Users) Proto() []*userv1.User {
	out := make([]*userv1.User, len(users))
	for i, u := range users {
		out[i] = u.Proto()
	}
	return out
}

// ExternalSessions provides an integration point for an external service to issue JWTs to control
// access to the cluster.
type ExternalSessions struct {
	LoginURI  string `json:"login_uri"`
	LogoutURI string `json:"logout_uri"`
	JwtKey    string `json:"jwt_key"`
}

// Enabled returns whether or not external sessions are enabled.
func (e ExternalSessions) Enabled() bool {
	return len(e.LoginURI) > 1
}

// UserWebSetting is a record of user web setting.
type UserWebSetting struct {
	UserID      UserID
	Key         string
	Value       string
	StoragePath string
}
