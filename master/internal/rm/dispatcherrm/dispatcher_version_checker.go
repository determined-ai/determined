package dispatcherrm

import (
	"context"
	"fmt"

	semvar "github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var launcherMinimumVersion = semvar.MustParse("3.3.1")

// Do a single check of the version.  Return an error
// if version cannot be obtained, or is below minimum.
func checkVersionNow(ctx context.Context,
	log *logrus.Entry,
	cl *launcherAPIClient,
) error {
	// The logger we will pass to the API client, so that when the API client
	// logs a message, we know who called it.
	launcherAPILogger := log.WithField("caller", "checkVersionNow")

	v, err := cl.getVersion(ctx, launcherAPILogger)
	if err != nil {
		return errors.Wrap(err, "cannot get launcher version")
	}

	if !checkLauncherVersion(v) {
		return fmt.Errorf("launcher version %s does not meet the required minimum. "+
			"Upgrade to hpe-hpc-launcher version %s or greater",
			v, launcherMinimumVersion)
	}

	log.Infof("HPC Launcher version %s", v)
	return nil
}

func checkLauncherVersion(v *semvar.Version) bool {
	return v.Equal(launcherMinimumVersion) || v.GreaterThan(launcherMinimumVersion)
}
