package db

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// DB is an interface for _all_ the functionality packed into the DB.
type DB interface {
	Migrate(migrationURL, codeURL string, actions []string) error
	Close() error
	GetOrCreateClusterID(telemetryID string) (string, error)
	AddExperiment(experiment *model.Experiment, modelDef []byte, activeConfig expconf.ExperimentConfig) error
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
	ExperimentNumTrials(id int) (int64, error)
	ExperimentTrialIDs(expID int) ([]int, error)
	ExperimentModelDefinitionRaw(id int) ([]byte, error)
	UpdateTrialFields(id int, newRunnerMetadata *trialv1.TrialRunnerMetadata, newRunID, newRestarts int) error
	TrialRunIDAndRestarts(trialID int) (int, int, error)
	AddTrainingMetrics(ctx context.Context, m *trialv1.TrialMetrics) error
	AddValidationMetrics(
		ctx context.Context, m *trialv1.TrialMetrics,
	) error
	ValidationByTotalBatches(trialID, totalBatches int) (*model.TrialMetrics, error)
	CheckpointByTotalBatches(trialID, totalBatches int) (*model.Checkpoint, error)
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
		labels *trialv1.TrialProfilerMetricLabels, offset, limit int,
	) (model.TrialProfilerMetricsBatchBatch, error)
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

	// ErrDeleteDefaultBinding is returned when trying to delete a workspace bound to its default
	// namespace.
	ErrDeleteDefaultBinding = errors.New("cannot delete the default namespace binding")
)
