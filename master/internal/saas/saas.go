package saas

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

type Provisioner interface {
	GetAndMaybeProvisionUserByToken(ctx context.Context, tokenText string,
		ext *model.ExternalSessions,
	) (*model.User, *model.UserSession, error)
}

var provisioners = map[string]Provisioner{}

func RegisterProvisioner(s string, p Provisioner) {
	provisioners[s] = p
}

func GetAndMaybeProvisionUserByToken(ctx context.Context, tokenText string,
	ext *model.ExternalSessions,
) (*model.User, *model.UserSession, error) {
	p, ok := provisioners["saas"]
	if !ok {
		panic("User provisioner for saas not initialized")
	}
	return p.GetAndMaybeProvisionUserByToken(ctx, tokenText, ext)
}
