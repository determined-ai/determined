package saas

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// Provisioner is an interface for provisioning external users by token.
type Provisioner interface {
	GetAndMaybeProvisionUserByToken(ctx context.Context, tokenText string,
		ext *model.ExternalSessions,
	) (*model.User, *model.UserSession, error)
}

var provisioners = map[string]Provisioner{}

// RegisterProvisioner registers an external user provisioner.
func RegisterProvisioner(s string, p Provisioner) {
	provisioners[s] = p
}

// GetAndMaybeProvisionUserByToken returns a user session derived from an external authentication token.
func GetAndMaybeProvisionUserByToken(ctx context.Context, tokenText string,
	ext *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	p, ok := provisioners["saas"]
	if !ok {
		panic("User provisioner for saas not initialized")
	}
	return p.GetAndMaybeProvisionUserByToken(ctx, tokenText, ext)
}
