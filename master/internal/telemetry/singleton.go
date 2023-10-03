package telemetry

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/pkg/config"
)

var (
	// defaultTelemeter is the global telemetry singleton.
	defaultTelemeter *telemeter
	syslog           = logrus.WithField("component", "telemetry")
)

// Init sets up the Telemetry singleton.
func Init(clusterID string, conf config.TelemetryConfig) {
	if defaultTelemeter != nil {
		syslog.Warn("detected re-initialization of Telemetry singleton that should never occur outside of tests")
		return
	}

	if !conf.Enabled || conf.SegmentMasterKey == "" {
		syslog.Info("telemetry reporting is disabled")
		return
	}
	syslog.Info("telemetry reporting is enabled; run with --telemetry-enabled=false to disable")

	client, err := analytics.NewWithConfig(
		conf.SegmentMasterKey,
		analytics.Config{Logger: debugLogger{}},
	)
	if err != nil {
		syslog.WithError(err).Warn("failed to initialize telemetry client")
		return
	}

	telemeter, err := newTelemeter(client, clusterID)
	if err != nil {
		syslog.WithError(err).Warn("failed to initialize telemetry service")
		return
	}
	defaultTelemeter = telemeter
}
