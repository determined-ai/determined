package user

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserByExternalToken returns a user session derived from an external authentication token.
func UserByExternalToken(tokenText string,
	ext *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	return nil, nil, nil
}
