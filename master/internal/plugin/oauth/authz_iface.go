package oauth

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// OauthAuthZ describes authz methods for experiments.
type OauthAuthZ interface {
	// CanAdministrateOauth returns an error if the user is not authorized to manage oauth.
	CanAdministrateOauth(ctx context.Context, curUser model.User) error
}

// AuthZProvider is the authz registry for experiments.
var AuthZProvider authz.AuthZProviderType[OauthAuthZ]
