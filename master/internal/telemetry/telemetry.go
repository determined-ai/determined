package telemetry

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/config"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	minTickIntervalMins = 10
	maxTickIntervalMins = 60
)

// MockTelemetry TBD, but putting this here for now to export to other tests.
func MockTelemetry() {
	mockRM := &mocks.ResourceManager{}
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).Return(
		&apiv1.GetResourcePoolsResponse{ResourcePools: []*resourcepoolv1.ResourcePool{}},
		nil,
	)
	mockDB := &mocks.DB{}
	mockDB.On("PeriodicTelemetryInfo").Return([]byte(`{"master_version": 1}`), nil)
	mockDB.On("CompleteAllocationTelemetry", mock.Anything).Return([]byte(`{"allocation_id": 1}`), nil)
	InitTelemetry(actor.NewSystem("Testing"), mockDB, mockRM, "1",
		config.TelemetryConfig{Enabled: true, SegmentMasterKey: "Test"},
	)
}

// telemetryRPFetcher exists mainly to avoid an annoying import cycle.
type telemetryRPFetcher interface {
	GetResourcePools(
		actor.Messenger,
		*apiv1.GetResourcePoolsRequest,
	) (*apiv1.GetResourcePoolsResponse, error)
}

// TelemetryActor manages gathering and sending telemetry data.
type TelemetryActor struct {
	db        db.DB
	rm        telemetryRPFetcher
	client    analytics.Client
	clusterID string
	syslog    *logrus.Entry
}

// DefaultTelemetry is the global telemetry singleton.
var DefaultTelemetry *TelemetryActor

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

	syslog := logrus.WithFields(logrus.Fields{
		"component":  fmt.Sprintf("telemetry-actor"),
		"clusterID":  clusterID,
		"segmentKey": segmentKey,
	})

	return &TelemetryActor{
		db:        db,
		rm:        rm,
		client:    client,
		clusterID: clusterID,
		syslog:    syslog,
	}, nil
}

// InitTelemetry sets up the actor for the telemetry.
func InitTelemetry(
	system *actor.System,
	db db.DB,
	rm telemetryRPFetcher,
	clusterID string,
	conf config.TelemetryConfig,
) {
	if DefaultTelemetry != nil {
		logrus.Warn(
			"detected re-initialization of Telemetry actor that should never occur outside of tests",
		)
	}

	if conf.Enabled && conf.SegmentMasterKey != "" {
		if actorDef, tErr := New(
			db,
			rm,
			clusterID,
			conf.SegmentMasterKey,
		); tErr != nil {
			logrus.WithError(tErr).Errorf("failed to initialize telemetry")
		} else {
			DefaultTelemetry = actorDef
			DefaultTelemetry.syslog.Info(
				"telemetry reporting is enabled; run with `--telemetry-enabled=false` to disable",
			)
			DefaultTelemetry.telemetryTick(system, 0)
		}
	} else {
		logrus.Info("telemetry reporting is disabled")
	}
}

// Track adds track call objects to the analytics.Client interface.
func (s *TelemetryActor) Track(t analytics.Track) {
	// Panic if telemetry isn't initialized or has crashed.
	if s == nil {
		panic("telemetry actor should not be nil: can't track.")
	}
	s.syslog.Infof("Tracking %s", t.Event)

	t.UserId = s.clusterID
	if err := s.client.Enqueue(t); err != nil {
		s.syslog.WithError(err).Warnf("failed to enqueue track %s", t.Event)
	}
}

func (s *TelemetryActor) telemetryTick(system *actor.System, t int) {
	// Panic if telemetry isn't initialized or has crashed.
	if s == nil {
		panic("telemetry actor should not be nil: can't tick.")
	}

	time.AfterFunc(time.Duration(t)*time.Minute, func() {
		resp, err := s.rm.GetResourcePools(system, &apiv1.GetResourcePoolsRequest{})
		if err != nil {
			// TODO(Brad): Make this routine more accepting of failures.
			s.syslog.WithError(err).Error("failed to receive resource pool telemetry information")
			return
		}
		// After waiting t minutes, report the first tick.
		go ReportMasterTick(resp, s.db)

		// Now call the next tick.
		bg := big.NewInt(maxTickIntervalMins - minTickIntervalMins)
		randNum, err := rand.Int(rand.Reader, bg)
		if err != nil {
			panic(err)
		}
		randInt := int(randNum.Int64()) + minTickIntervalMins
		s.telemetryTick(system, randInt)
	})
}
