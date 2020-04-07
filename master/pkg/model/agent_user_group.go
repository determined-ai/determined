package model

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/archive"
)

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

// OwnedArchiveItem will create an archive.Item owned by the AgentUserGroup, or by root if c is nil.
func (c *AgentUserGroup) OwnedArchiveItem(
	path string, content []byte, mode int, fileType byte,
) archive.Item {
	if c == nil {
		return archive.RootItem(path, content, mode, fileType)
	}
	return archive.UserItem(path, content, mode, fileType, c.UID, c.GID)
}

// OwnArchive will return an archive.Archive modified to be owned by the AgentUserGroup, or
// unmodified if c is nil.
func (c *AgentUserGroup) OwnArchive(oldArchive archive.Archive) archive.Archive {
	if c == nil {
		return oldArchive
	}
	var newArchive archive.Archive
	for _, item := range oldArchive {
		newItem := item
		newItem.UserID = c.UID
		newItem.GroupID = c.GID
		newArchive = append(newArchive, newItem)
	}
	return newArchive
}
