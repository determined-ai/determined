package telemetry

import (
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/config"
)

// Setup sets up the actor for the telemetry.
func Setup(
	system *actor.System,
	db db.DB,
	rm telemetryRPFetcher,
	clusterID string,
	conf config.TelemetryConfig,
) {
	if conf.Enabled && conf.SegmentMasterKey != "" {
		if actorDef, tErr := New(
			db,
			rm,
			clusterID,
			conf.SegmentMasterKey,
		); tErr != nil {
			log.WithError(tErr).Errorf("failed to initialize telemetry")
		} else {
			log.Info("telemetry reporting is enabled; run with `--telemetry-enabled=false` to disable")
			system.ActorOf(actor.Addr("telemetry"), actorDef)
		}
	} else {
		log.Info("telemetry reporting is disabled")
	}
}
