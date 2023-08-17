package telemetry

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/config"
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

// Telemeter manages gathering and sending telemetry data.
type Telemeter struct {
	db        db.DB
	rm        telemetryRPFetcher
	client    analytics.Client
	clusterID string
	syslog    *logrus.Entry
}

// DefaultTelemeter is the global telemetry singleton.
var DefaultTelemeter *Telemeter

// New initializes a Telemetry struct and returns it. Can error on Segment client init.
func New(
	db db.DB,
	rm telemetryRPFetcher,
	clusterID string,
	segmentKey string,
) (*Telemeter, error) {
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
		"component":  fmt.Sprintf("telemetry"),
		"clusterID":  clusterID,
		"segmentKey": segmentKey,
	})

	return &Telemeter{
		db:        db,
		rm:        rm,
		client:    client,
		clusterID: clusterID,
		syslog:    syslog,
	}, nil
}

// Init sets up the Telemetry singleton.
func Init(
	system *actor.System,
	db db.DB,
	rm telemetryRPFetcher,
	clusterID string,
	conf config.TelemetryConfig,
) {
	if DefaultTelemeter != nil {
		logrus.Warn(`detected re-initialization of Telemetry singleton `,
			`that should never occur outside of tests`)
		return
	}

	if !conf.Enabled || conf.SegmentMasterKey == "" {
		logrus.Info("telemetry reporting is disabled")
		return
	}

	telemetryDef, err := New(
		db,
		rm,
		clusterID,
		conf.SegmentMasterKey,
	)
	if err != nil {
		logrus.WithError(err).Errorf("failed to initialize telemetry")
		return
	}

	DefaultTelemeter = telemetryDef
	DefaultTelemeter.syslog.Info(`telemetry reporting is enabled; `,
		`run with --telemetry-enabled=false to disable`)
	go DefaultTelemeter.tick(system)
}

// track adds track call objects to the analytics.Client interface.
func (s *Telemeter) track(t analytics.Track) {
	if s == nil {
		return
	}

	t.UserId = s.clusterID
	if err := s.client.Enqueue(t); err != nil {
		s.syslog.WithError(err).WithField("event", t.Event).Warn("failed to enqueue track")
	}
}

func (s *Telemeter) tick(system *actor.System) {
	if s == nil {
		return
	}

	for {
		resp, err := s.rm.GetResourcePools(system, &apiv1.GetResourcePoolsRequest{})
		if err != nil {
			// TODO(Brad): Make this routine more accepting of failures.
			s.syslog.WithError(err).Error("failed to receive resource pool telemetry information")
			return
		}
		ReportMasterTick(resp, s.db)
		time.Sleep(s.sleepInterval())
	}
}

func (s *Telemeter) sleepInterval() time.Duration {
	bg := big.NewInt(maxTickIntervalMins - minTickIntervalMins)
	randNum, err := rand.Int(rand.Reader, bg)
	if err != nil {
		s.syslog.Error(err)
		return time.Duration(maxTickIntervalMins) * time.Minute
	}
	randInt := int(randNum.Int64()) + minTickIntervalMins
	return time.Duration(randInt) * time.Minute
}
