package dispatcherrm

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"gotest.tools/assert"
)

func TestCheckLauncherVersion(t *testing.T) {
	assert.Equal(t, checkLauncherVersion(semver.MustParse("4.1.0")), true)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("4.1.3-SNAPSHOT")), true)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("3.2.4")), true)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("3.2.3")), false)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("3.2.0")), false)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("3.1.3")), false)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("3.1.0")), false)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("2.3.3")), false)
	assert.Equal(t, checkLauncherVersion(semver.MustParse("3.0.3")), false)
}
