package provisioner

import (
	"crypto/tls"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// Setup initializes the provisioner.
func Setup(
	ctx *actor.Context,
	config *provconfig.Config,
	resourcePool string,
	cert *tls.Certificate,
	db db.DB,
) (*Provisioner, error) {
	ctx.Log().Info("found provisioner configuration")
	if config.AWS != nil {
		ctx.Log().Info("connecting to AWS")
	}
	if config.GCP != nil {
		ctx.Log().Info("connecting to GCP")
	}
	provisioner, err := New(ctx.Self().System(), resourcePool, config, cert, db)
	if err != nil {
		return nil, errors.Wrap(err, "error creating provisioner")
	}
	go provisioner.Run()
	return provisioner, nil
}
