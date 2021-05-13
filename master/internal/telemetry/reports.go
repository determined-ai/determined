package telemetry

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
		"id":               e.ID,
		"searcher":         e.Config.Searcher,
		"resources":        e.Config.Resources,
		"image":            e.Config.Environment.Image,
		"num_hparams":      len(e.Config.Hyperparameters),
		"batches_per_step": e.Config.SchedulingUnit,
	})
}

func fetchNumTrials(db *db.PgDB, experimentID int) *int64 {
	result, err := db.ExperimentNumTrials(experimentID)
	if err != nil {
		logrus.WithError(err).Warn("failed to fetch telemetry metrics")
		return nil
	}
	return &result
}

func fetchNumSteps(db *db.PgDB, experimentID int) *int64 {
	result, err := db.ExperimentNumSteps(experimentID)
	if err != nil {
		logrus.WithError(err).Warn("failed to fetch telemetry metrics")
		return nil
	}
	return &result
}

func fetchTotalStepTime(db *db.PgDB, experimentID int) *float64 {
	result, err := db.ExperimentTotalStepTime(experimentID)
	if err != nil {
		logrus.WithError(err).Warn("failed to fetch telemetry metrics")
		return nil
	}
	return &result
}

// ReportExperimentStateChanged reports that the state of an experiment has changed.
func ReportExperimentStateChanged(system *actor.System, db *db.PgDB, e model.Experiment) {
	var numTrials *int64
	var numSteps *int64
	var totalStepTime *float64

	if model.TerminalStates[e.State] {
		// Report additional metrics when an experiment reaches a terminal state.
		// These metrics are null for non-terminal state transitions.
		numTrials = fetchNumTrials(db, e.ID)
		numSteps = fetchNumSteps(db, e.ID)
		totalStepTime = fetchTotalStepTime(db, e.ID)
	}

	report(system, "experiment_state_changed", map[string]interface{}{
		"id":              e.ID,
		"state":           e.State,
		"start_time":      e.StartTime,
		"end_time":        e.EndTime,
		"num_trials":      numTrials,
		"num_steps":       numSteps,
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

// ReportResourcePoolCreated reports that a resource pool has been created.
func ReportResourcePoolCreated(
	system *actor.System,
	poolName, schedulerType,
	fittingPolicy string,
	preemptionEnabled bool,
) {
	report(system, "resource_pool_created", map[string]interface{}{
		"pool_name":          poolName,
		"scheduler_type":     schedulerType,
		"fitting_policy":     fittingPolicy,
		"preemption_enabled": preemptionEnabled,
		"version":            "1",
	})
}
