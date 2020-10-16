package telemetry

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/check"
)

// telemetryActor manages gathering and sending telemetry data.
type telemetryActor struct {
	db           *db.PgDB
	client       analytics.Client
	tickInterval time.Duration
	clusterID    string
}

type telemetryTick struct {
	cause string
}

// debugLogger is an implementation of Segment's logger type that prints all messages at the debug
// level in order to reduce noise from failed messages.
type debugLogger struct{}

// Logf implements the analytics.Logger interface.
func (debugLogger) Logf(s string, a ...interface{}) {
	logrus.Debugf("segment log message: "+s, a...)
}

// Errorf implements the analytics.Logger interface.
func (debugLogger) Errorf(s string, a ...interface{}) {
	logrus.Debugf("segment error message: "+s, a...)
}

// NewActor creates an actor to handle collecting and sending telemetry information.
func NewActor(
	db *db.PgDB,
	clusterID string,
	masterID string,
	masterVersion string,
	resourceManagerType string,
	segmentKey string,
) (actor.Actor, error) {
	client, err := analytics.NewWithConfig(
		segmentKey,
		analytics.Config{Logger: debugLogger{}},
	)
	if err != nil {
		return nil, err
	}

	err = client.Enqueue(analytics.Identify{
		UserId: clusterID,
		Traits: analytics.Traits{
			"go_version":            runtime.Version(),
			"master_id":             masterID,
			"master_version":        masterVersion,
			"resource_manager_type": resourceManagerType,
		},
	})
	if err != nil {
		logrus.WithError(err).Warn("failed to identify for telemetry")
	}

	return &telemetryActor{db, client, 1 * time.Hour, clusterID}, nil
}

func (s *telemetryActor) enqueue(ctx *actor.Context, t analytics.Track) {
	check.Panic(check.Equal(t.UserId, ""))
	t.UserId = s.clusterID
	if err := s.client.Enqueue(t); err != nil {
		ctx.Log().WithError(err).Warnf("failed to enqueue event %s", t.Event)
	}
}

func (s *telemetryActor) snapshotValues() (analytics.Properties, error) {
	dbInfo, err := s.db.PeriodicTelemetryInfo()
	if err != nil {
		return nil, err
	}

	props := analytics.Properties{}
	if err = json.Unmarshal(dbInfo, &props); err != nil {
		return nil, err
	}
	return props, nil
}

// Receive implements the actor.Actor interface.
func (s *telemetryActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, 0, telemetryTick{"master_started"})

	case analytics.Track:
		s.enqueue(ctx, msg)

	case telemetryTick:
		actors.NotifyAfter(ctx, s.tickInterval, telemetryTick{"master_tick"})

		props, err := s.snapshotValues()
		if err != nil {
			// Log the error but return nil so that this actor continues running.
			ctx.Log().WithError(err).Error("failed to retrieve telemetry information")
			return nil
		}
		s.enqueue(ctx, analytics.Track{
			Event:      msg.cause,
			Properties: props,
		})

	case actor.PostStop:
		s.enqueue(ctx, analytics.Track{
			Event: "master_stopped",
		})
		_ = s.client.Close()
	}

	return nil
}
