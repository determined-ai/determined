package telemetry

import (
	"github.com/google/uuid"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

func report(system *actor.System, event string, properties map[string]interface{}) {
	system.TellAt(
		actor.Addr("telemetry"),
		analytics.Track{Event: event, Properties: analytics.Properties(properties)},
	)
}

// ReportAgentConnected reports that an agent has connected to the master.
func ReportAgentConnected(system *actor.System, uuid uuid.UUID, devices []device.Device) {
	report(system, "agent_connected", map[string]interface{}{
		"uuid":    uuid,
		"devices": devices,
	})
}

// ReportAgentDisconnected reports that an agent has discconnected from the master.
func ReportAgentDisconnected(system *actor.System, uuid uuid.UUID) {
	report(system, "agent_disconnected", map[string]interface{}{
		"uuid": uuid,
	})
}

// ReportExperimentCreated reports that an experiment has been created.
func ReportExperimentCreated(system *actor.System, e model.Experiment) {
	report(system, "experiment_created", map[string]interface{}{
		"id":        e.ID,
		"searcher":  e.Config.Searcher,
		"resources": e.Config.Resources,
	})
}

// ReportExperimentStateChanged reports that the state of an experiment has changed.
func ReportExperimentStateChanged(system *actor.System, db *db.PgDB, e model.Experiment) {
	var totalStepTime *float64
	if model.TerminalStates[e.State] {
		if t, err := db.ExperimentTotalStepTime(e.ID); err == nil {
			seconds := t.Seconds()
			totalStepTime = &seconds
		}
	}
	report(system, "experiment_state_changed", map[string]interface{}{
		"id":              e.ID,
		"state":           e.State,
		"start_time":      e.StartTime,
		"end_time":        e.EndTime,
		"total_step_time": totalStepTime,
	})
}

// ReportUserCreated reports that a user has been created.
func ReportUserCreated(system *actor.System, admin, active bool) {
	report(system, "user_created", map[string]interface{}{
		"admin":  admin,
		"active": active,
	})
}
