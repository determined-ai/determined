package provisioner

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Setup initializes and registers the actor for the provisioner.
func Setup(system *actor.System, config *Config) (*Provisioner, *actor.Ref, error) {
	if config == nil {
		log.Info("cannot find provisioner configuration, disabling provisioner")
		return nil, nil, nil
	}
	log.Info("found provisioner configuration")
	if config.AWS != nil {
		log.Info("connecting to AWS")
	}
	if config.GCP != nil {
		log.Info("connecting to GCP")
	}
	provisioner, err := New(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error creating provisioner")
	}
	provisionerActor, _ := system.ActorOf(actor.Addr("provisioner"), provisioner)
	return provisioner, provisionerActor, nil
}
