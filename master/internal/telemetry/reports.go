package telemetry

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"reflect"
	"time"

	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/version"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/devicev1"
)

const (
	minTickIntervalMins = 10
	maxTickIntervalMins = 60
)

// telemetryRPFetcher exists mainly to avoid an annoying import cycle.
type telemetryRPFetcher interface {
	GetResourcePools(
		*apiv1.GetResourcePoolsRequest,
	) (*apiv1.GetResourcePoolsResponse, error)
}

// PeriodicallyReportMasterTick periodically reports various telemetry information about the
// running master. It should be called once per cluster.
func PeriodicallyReportMasterTick(db db.DB, rm telemetryRPFetcher, system *actor.System) {
	if defaultTelemeter == nil {
		return
	}

	for {
		reportMasterTick(db, rm, system)
		time.Sleep(reportMasterTickDelay())
	}
}

func reportMasterTickDelay() time.Duration {
	bg := big.NewInt(maxTickIntervalMins - minTickIntervalMins)
	randNum, err := rand.Int(rand.Reader, bg)
	if err != nil {
		syslog.Error(err)
		return time.Duration(maxTickIntervalMins) * time.Minute
	}
	randInt := int(randNum.Int64()) + minTickIntervalMins
	return time.Duration(randInt) * time.Minute
}

// reportMasterTick reports the master snapshot on a periodic tick.
func reportMasterTick(db db.DB, rm telemetryRPFetcher, system *actor.System) {
	resp, err := rm.GetResourcePools(&apiv1.GetResourcePoolsRequest{})
	if err != nil {
		// TODO(Brad): Make this routine more accepting of failures.
		syslog.WithError(err).Error("failed to receive resource pool telemetry information")
		return
	}

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
		syslog.WithError(err).Error("failed to retrieve telemetry information")
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
		syslog.WithError(err).Error("failed to retrieve telemetry information")
		return
	}

	defaultTelemeter.track(analytics.Track{
		Event:      "master_tick",
		Properties: props,
	})
}

// ReportProvisionerTick reports the state of all provision requests by a provisioner.
func ReportProvisionerTick(instances []*model.Instance, instanceType string) {
	defaultTelemeter.track(
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
	defaultTelemeter.track(
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
		syslog.WithError(err).Warn("failed to fetch allocation telemetry")
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
		syslog.WithError(err).Warn("failed to report allocation telemetry")
		return
	}

	defaultTelemeter.track(
		analytics.Track{
			Event:      "allocation_terminal",
			Properties: props,
		},
	)
}

func fetchNumTrials(db db.DB, experimentID int) *int64 {
	result, err := db.ExperimentNumTrials(experimentID)
	if err != nil {
		syslog.WithError(err).Warn("failed to fetch telemetry metrics")
		return nil
	}
	return &result
}

func fetchNumSteps(db db.DB, experimentID int) *int64 {
	result, err := db.ExperimentNumSteps(experimentID)
	if err != nil {
		syslog.WithError(err).Warn("failed to fetch telemetry metrics")
		return nil
	}
	return &result
}

func fetchTotalStepTime(db db.DB, experimentID int) *float64 {
	result, err := db.ExperimentTotalStepTime(experimentID)
	if err != nil {
		syslog.WithError(err).Warn("failed to fetch telemetry metrics")
		return nil
	}
	return &result
}

// ReportExperimentStateChanged reports that the state of an experiment has changed.
func ReportExperimentStateChanged(db db.DB, e *model.Experiment) {
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

	defaultTelemeter.track(
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
	defaultTelemeter.track(
		analytics.Track{
			Event: "user_created",
			Properties: map[string]interface{}{
				"admin":  admin,
				"active": active,
			},
		},
	)
}
