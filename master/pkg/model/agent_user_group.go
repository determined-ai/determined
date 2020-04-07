package model

import "github.com/pkg/errors"

// An AgentUserGroup represents a username and primary group for a user on an
// agent host machine. There is at most one AgentUserGroup for each User.
type AgentUserGroup struct {
	ID int `db:"id" json:"id"`

	UserID UserID `db:"user_id" json:"user_id"`

	// The User is the username on an agent host machine. This may be different
	// from the username of the user in the User database.
	User string `db:"user_" json:"user"`
	UID  int    `db:"uid" json:"uid"`

	// The Group is the primary group of the user.
	Group string `db:"group_" json:"group"`
	GID   int    `db:"gid" json:"gid"`
}

// Validate validates the fields of the AgentUserGroup.
func (c AgentUserGroup) Validate() []error {
	var errs []error

	if c.UID < 0 {
		errs = append(errs, errors.New("uid less than zero"))
	}

	if c.GID < 0 {
		errs = append(errs, errors.New("gid less than zero"))
	}

	if len(c.User) == 0 {
		errs = append(errs, errors.New("user not set"))
	}

	if len(c.Group) == 0 {
		errs = append(errs, errors.New("group not set"))
	}

	return errs
}
