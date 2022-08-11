package telemetry

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	minTickIntervalMins = 10
	maxTickIntervalMins = 60
)

// telemetryRPFetcher exists mainly to avoid an annoying import cycle.
type telemetryRPFetcher interface {
	GetResourcePools(
		actor.Messenger,
		*apiv1.GetResourcePoolsRequest,
	) (*apiv1.GetResourcePoolsResponse, error)
}

type telemetryTick struct{}

// TelemetryActor manages gathering and sending telemetry data.
type TelemetryActor struct {
	db        db.DB
	rm        telemetryRPFetcher
	client    analytics.Client
	clusterID string
}

// New creates an actor to handle collecting and sending telemetry information.
func New(
	db db.DB,
	rm telemetryRPFetcher,
	clusterID string,
	segmentKey string,
) (*TelemetryActor, error) {
	client, err := analytics.NewWithConfig(
		segmentKey,
		analytics.Config{Logger: debugLogger{}},
	)
	if err != nil {
		return nil, err
	}

	if err := client.Enqueue(analytics.Identify{
		UserId: clusterID,
		Traits: analytics.Traits{
			"master_version": version.Version,
		},
	}); err != nil {
		logrus.WithError(err).Warnf("failed to enqueue identity %s", clusterID)
	}

	return &TelemetryActor{db, rm, client, clusterID}, nil
}

// Receive implements the actor.Actor interface.
func (s *TelemetryActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, 0, telemetryTick{})

	case analytics.Track:
		msg.UserId = s.clusterID
		if err := s.client.Enqueue(msg); err != nil {
			ctx.Log().WithError(err).Warnf("failed to enqueue track %s", msg.Event)
		}

	case telemetryTick:
		// Tick in a random interval.
		//nolint:gosec // Weak RNG is fine here.
		randNum := rand.Intn(maxTickIntervalMins-minTickIntervalMins) + minTickIntervalMins
		actors.NotifyAfter(ctx, time.Duration(randNum)*time.Minute, telemetryTick{})

		ReportMasterTick(ctx.Self().System(), s.db, s.rm)

	case actor.PostStop:
		_ = s.client.Close()
	}

	return nil
}
