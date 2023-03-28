package dispatcherrm

import (
	"context"
	"time"

	semvar "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
)

const versionCheckPeriod = 60

var launcherMinimumVersion = semvar.MustParse("3.2.4")

// periodicallyCheckLauncherVersion checks the launcher version every 60s, logging warnings while
// it is out of date and exiting if it finds it is ok.
func periodicallyCheckLauncherVersion(
	ctx context.Context,
	log *logrus.Entry,
	cl *launcherAPIClient,
) {
	for range time.NewTicker(versionCheckPeriod).C {
		v, err := cl.getVersion(ctx)
		if err != nil {
			log.WithError(err).Error("could not get launcher API version")
			continue
		}

		if checkLauncherVersion(v) {
			return
		}

		log.Errorf("Launcher version %s does not meet the required minimum. "+
			"Upgrade to hpe-hpc-launcher version %s",
			v, launcherMinimumVersion)
	}
}

func checkLauncherVersion(v *semvar.Version) bool {
	return v.Equal(launcherMinimumVersion) || v.GreaterThan(launcherMinimumVersion)
}
