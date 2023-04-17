package provisioner

import (
	"crypto/tls"
	"fmt"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// Setup initializes and registers the actor for the provisioner.
func Setup(
	ctx *actor.Context,
	config *provconfig.Config,
	resourcePool string,
	cert *tls.Certificate,
	db db.DB,
) (*Provisioner, *actor.Ref, error) {
	ctx.Log().Info("found provisioner configuration")
	if config.AWS != nil {
		ctx.Log().Info("connecting to AWS")
	}
	if config.GCP != nil {
		ctx.Log().Info("connecting to GCP")
	}
	provisioner, err := New(resourcePool, config, cert, db)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating provisioner: %w", err)
	}
	provisionerActor, _ := ctx.ActorOf("provisioner", provisioner)
	return provisioner, provisionerActor, nil
}
