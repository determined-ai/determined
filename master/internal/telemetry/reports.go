package telemetry

import (
	"encoding/json"
	"reflect"

	"github.com/sirupsen/logrus"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
)

// ReportMasterTick reports the master snapshot on a periodic tick.
func ReportMasterTick(resp *apiv1.GetResourcePoolsResponse, db db.DB) {
	resourceManagerType := ""

	gpuTotalNum, gpuUsedNum := 0, 0
	poolTypes := make(map[string]int, len(resp.ResourcePools))
	for _, pool := range resp.ResourcePools {
		poolTypes[sproto.StringFromResourcePoolTypeProto(pool.Type)]++
		if pool.SlotType == devicev1.Type_TYPE_CUDA || pool.SlotType == devicev1.Type_TYPE_ROCM {
			gpuTotalNum += int(pool.SlotsAvailable)
			gpuUsedNum += int(pool.SlotsUsed)
		}
	}

	dbInfo, err := db.PeriodicTelemetryInfo()
	if err != nil {
		logrus.WithError(err).Error("failed to retrieve telemetry information")
		return
	}

	props := analytics.Properties{
		"master_version":        version.Version,
		"resource_manager_type": resourceManagerType,
		"pool_type":             poolTypes,
		"gpu_total_num":         gpuTotalNum,
		"gpu_used_num":          gpuUsedNum,
	}

	if err = json.Unmarshal(dbInfo, &props); err != nil {
		logrus.WithError(err).Error("failed to retrieve telemetry information")
		return
	}

	DefaultTelemetry.Track(
		analytics.Track{
			Event:      "master_tick",
			Properties: props,
		})
}

// ReportProvisionerTick reports the state of all provision requests by a provisioner.
func ReportProvisionerTick(instances []*model.Instance, instanceType string) {
	DefaultTelemetry.Track(
		analytics.Track{
			Event: "provisioner_tick",
			Properties: map[string]interface{}{
				"instance_type": instanceType,
				"instances":     instances,
			},
		})
}

// ReportExperimentCreated reports that an experiment has been created.
func ReportExperimentCreated(id int, config expconf.ExperimentConfig) {
	DefaultTelemetry.Track(
		analytics.Track{
			Event: "experiment_created",
			Properties: map[string]interface{}{
				"id":                        id,
				"num_hparams":               len(config.Hyperparameters()),
				"resources_slots_per_trial": config.Resources().SlotsPerTrial(),
				"image":                     config.Environment().Image(),

				"searcher_name": reflect.TypeOf(config.Searcher().GetUnionMember()),
			},
		},
	)
}

// ReportAllocationTerminal reports that an allocation ends.
func ReportAllocationTerminal(db db.DB, a model.Allocation, d *device.Device,
) {
	res, err := db.CompleteAllocationTelemetry(a.AllocationID)
	if err != nil {
		logrus.WithError(err).Warn("failed to fetch allocation telemetry")
		return
	}

	props := analytics.Properties{
		"allocation_id": a.AllocationID,
		"task_id":       a.TaskID,
		"start_time":    a.StartTime,
		"end_time":      a.EndTime,
		"slots":         a.Slots,
	}
	if d != nil {
		props["slot_type"] = d.Type
		props["slot_brand"] = d.Brand
	}

	if err = json.Unmarshal(res, &props); err != nil {
		logrus.WithError(err).Warn("failed to report allocation telemetry")
		return
	}

	DefaultTelemetry.Track(
		analytics.Track{
			Event:      "allocation_terminal",
			Properties: props,
		},
	)
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
func ReportExperimentStateChanged(db *db.PgDB, e model.Experiment) {
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

	DefaultTelemetry.Track(
		analytics.Track{
			Event: "experiment_state_changed",
			Properties: map[string]interface{}{
				"id":              e.ID,
				"state":           e.State,
				"start_time":      e.StartTime,
				"end_time":        e.EndTime,
				"num_trials":      numTrials,
				"num_steps":       numSteps,
				"total_step_time": totalStepTime,
			},
		},
	)
}

// ReportUserCreated reports that a user has been created.
func ReportUserCreated(admin, active bool) {
	DefaultTelemetry.Track(
		analytics.Track{
			Event: "user_created",
			Properties: map[string]interface{}{
				"admin":  admin,
				"active": active,
			},
		},
	)
}
