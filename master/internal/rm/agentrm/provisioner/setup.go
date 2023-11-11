package provisioner

import (
	"crypto/tls"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/internal/db"
)

var syslog = logrus.WithField("component", "provisioner")

// Setup initializes the provisioner.
func Setup(
	config *provconfig.Config,
	resourcePool string,
	cert *tls.Certificate,
	db db.DB,
) (*Provisioner, error) {
	syslog.Info("found provisioner configuration")
	if config.AWS != nil {
		syslog.Info("connecting to AWS")
	}
	if config.GCP != nil {
		syslog.Info("connecting to GCP")
	}
	provisioner, err := New(resourcePool, config, cert, db)
	if err != nil {
		return nil, errors.Wrap(err, "error creating provisioner")
	}
	go provisioner.Run()
	return provisioner, nil
}
