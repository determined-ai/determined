package telemetry

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/version"
)

// telemeter manages gathering and sending telemetry data.
type telemeter struct {
	client    analytics.Client
	clusterID string
	syslog    *logrus.Entry
}

// newTelemeter initializes a Telemetry struct and returns it. Can error on Segment client init.
func newTelemeter(client analytics.Client, clusterID string) (*telemeter, error) {
	if err := client.Enqueue(analytics.Identify{
		UserId: clusterID,
		Traits: analytics.Traits{
			"master_version": version.Version,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to enqueue identity %s: %w", clusterID, err)
	}

	return &telemeter{
		client:    client,
		clusterID: clusterID,
		syslog:    syslog.WithField("clusterID", clusterID),
	}, nil
}

// track adds track call objects to the analytics.Client interface.
func (s *telemeter) track(t analytics.Track) {
	if s == nil {
		return
	}

	t.UserId = s.clusterID
	if err := s.client.Enqueue(t); err != nil {
		s.syslog.WithError(err).WithField("event", t.Event).Warn("failed to enqueue track")
	}
}
