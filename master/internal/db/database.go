package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// DB is an interface for _all_ the functionality packed into the DB.
type DB interface {
	Migrate(migrationURL string, actions []string) error
	Close() error
	GetOrCreateClusterID(telemetryID string) (string, error)
	CheckExperimentExists(id int) (bool, error)
	CheckTrialExists(id int) (bool, error)
	TrialExperimentAndRequestID(id int) (int, model.RequestID, error)
	AddExperiment(experiment *model.Experiment, activeConfig expconf.ExperimentConfig) error
	ExperimentIDByTrialID(trialID int) (int, error)
	NonTerminalExperiments() ([]*model.Experiment, error)
	TerminateExperimentInRestart(id int, state model.State) error
	SaveExperimentConfig(id int, config expconf.ExperimentConfig) error
	SaveExperimentState(experiment *model.Experiment) error
	SaveExperimentArchiveStatus(experiment *model.Experiment) error
	DeleteExperiments(ctx context.Context, ids []int) error
	ExperimentHasCheckpointsInRegistry(id int) (bool, error)
	SaveExperimentProgress(id int, progress *float64) error
	ActiveExperimentConfig(id int) (expconf.ExperimentConfig, error)
	ExperimentTotalStepTime(id int) (float64, error)
	ExperimentNumTrials(id int) (int64, error)
	ExperimentTrialIDs(expID int) ([]int, error)
	ExperimentNumSteps(id int) (int64, error)
	ExperimentModelDefinitionRaw(id int) ([]byte, error)
	ExperimentCheckpointsToGCRaw(
		id int,
		experimentBest, trialBest, trialLatest int,
	) ([]uuid.UUID, error)
	AddTask(t *model.Task) error
	UpdateTrial(id int, newState model.State) error
	UpdateTrialRunnerState(id int, state string) error
	UpdateTrialRunnerMetadata(id int, md *trialv1.TrialRunnerMetadata) error
	AddAllocation(a *model.Allocation) error
	CompleteAllocation(a *model.Allocation) error
	CompleteAllocationTelemetry(aID model.AllocationID) ([]byte, error)
	TrialRunIDAndRestarts(trialID int) (int, int, error)
	UpdateTrialRunID(id, runID int) error
	UpdateTrialRestarts(id, restarts int) error
	AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error
	AddValidationMetrics(
		ctx context.Context, m *trialv1.TrialMetrics,
	) error
	ValidationByTotalBatches(trialID, totalBatches int) (*model.TrialMetrics, error)
	CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error)
	CheckpointByUUID(id uuid.UUID) (*model.Checkpoint, error)
	LatestCheckpointForTrial(trialID int) (*model.Checkpoint, error)
	PeriodicTelemetryInfo() ([]byte, error)
	TrialState(trialID int) (model.State, error)
	TrialStatus(trialID int) (model.State, *time.Time, error)
	Query(queryName string, v interface{}, params ...interface{}) error
	QueryF(
		queryName string, args []interface{}, v interface{}, params ...interface{}) error
	RawQuery(queryName string, params ...interface{}) ([]byte, error)
	UpdateResourceAllocationAggregation() error
	InsertTrialProfilerMetricsBatch(
		values []float32, batches []int32, timestamps []time.Time, labels []byte,
	) error
	GetTrialProfilerMetricsBatches(
		labelsJSON []byte, offset, limit int,
	) (model.TrialProfilerMetricsBatchBatch, error)
	ProjectByName(workspaceName string, projectName string) (projectID int, err error)
	ProjectExperiments(id int) (experiments []*model.Experiment, err error)
	ExperimentLabelUsage(projectID int32) (labelUsage map[string]int, err error)
	GetExperimentStatus(experimentID int) (state model.State, progress float64,
		err error)
	TrainingMetricBatches(experimentID int, metricName string, startTime time.Time) (
		batches []int32, endTime time.Time, err error)
	ValidationMetricBatches(experimentID int, metricName string, startTime time.Time) (
		batches []int32, endTime time.Time, err error)
	TrialsSnapshot(experimentID int, minBatches int, maxBatches int,
		metricName string, startTime time.Time, metricGroup model.MetricGroup) (
		trials []*apiv1.TrialsSnapshotResponse_Trial, endTime time.Time, err error)
	TopTrialsByTrainingLength(experimentID int, maxTrials int, metric string,
		smallerIsBetter bool) (trials []int32, err error)
	StartAllocationSession(allocationID model.AllocationID, owner *model.User) (string, error)
	DeleteAllocationSession(allocationID model.AllocationID) error
	UpdateAllocationState(allocation model.Allocation) error
	UpdateAllocationStartTime(allocation model.Allocation) error
	UpdateAllocationProxyAddress(allocation model.Allocation) error
	ExperimentSnapshot(experimentID int) ([]byte, int, error)
	SaveSnapshot(
		experimentID int, version int, experimentSnapshot []byte,
	) error
	DeleteSnapshotsForExperiment(experimentID int) error
	DeleteSnapshotsForTerminalExperiments() error
	QueryProto(queryName string, v interface{}, args ...interface{}) error
	QueryProtof(
		queryName string, args []interface{}, v interface{}, params ...interface{}) error
	TrialLogs(
		trialID, limit int, fs []api.Filter, order apiv1.OrderBy, followState interface{},
	) ([]*model.TrialLog, interface{}, error)
	DeleteTrialLogs(ids []int) error
	TrialLogsCount(trialID int, fs []api.Filter) (int, error)
	TrialLogsFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error)
	RecordAgentStats(a *model.AgentStats) error
	EndAllAgentStats() error
	RecordInstanceStats(a *model.InstanceStats) error
	EndInstanceStats(a *model.InstanceStats) error
	EndAllInstanceStats() error
	EndAllTaskStats() error
	RecordTaskEndStats(stats *model.TaskStats) error
	RecordTaskStats(stats *model.TaskStats) error
	UpdateJobPosition(jobID model.JobID, position decimal.Decimal) error
}

var (
	// ErrNotFound is returned if nothing is found.
	ErrNotFound = errors.New("not found")

	// ErrTooManyRowsAffected is returned if too many rows are affected.
	ErrTooManyRowsAffected = errors.New("too many rows are affected")

	// ErrDuplicateRecord is returned when trying to create a row that already exists.
	ErrDuplicateRecord = errors.New("row already exists")

	// ErrInvalidInput is returned when the data passed to a function is invalid for semantic or
	// syntactic reasons.
	ErrInvalidInput = errors.New("invalid input")
)
