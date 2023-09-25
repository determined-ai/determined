package gcp

import (
	"testing"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGCPNodeNameGenGreaterThanMaxLength(t *testing.T) {
	cluster := &gcpCluster{
		config: &provconfig.GCPClusterConfig{
			NamePrefix: "CpTVyfTKBqrZngPVErlsekl7pc2k4ZkwdaTeRK3l6wqDdHbNXYmCnwiQ3G8qzWld",
		},
		syslog: logrus.WithField("gcp-cluster", "resourcePool"),
	}
	name := cluster.generateInstanceNamePattern()
	assert.Equal(t, 63, len(name))
}
