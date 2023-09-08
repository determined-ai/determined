package trials

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	batches = "batches"
)

// MetricsTimeSeries returns a time-series of the specified metric in the specified
// trial.
func MetricsTimeSeries(trialID int32, startTime time.Time,
	metricNames []string,
	startBatches int, endBatches int, xAxisMetricLabels []string,
	maxDatapoints int, timeSeriesColumn string,
	timeSeriesFilter *commonv1.PolymorphicFilter, metricGroup model.MetricGroup) (
	metricMeasurements []db.MetricMeasurements, err error,
) {
	var queryColumn, orderColumn string
	// The data for batches and column are stored under different column names
	switch timeSeriesColumn {
	case "batches":
		queryColumn = "total_batches"
	case "time":
		queryColumn = "end_time"
	default:
		queryColumn = strings.ReplaceAll(timeSeriesColumn, ".", "·")
	}
	subq := db.BunSelectMetricsQuery(metricGroup, false).Table("metrics").
		ColumnExpr("(select setseed(1)) as _seed").
		ColumnExpr("total_batches as batches").
		ColumnExpr("trial_id").ColumnExpr("end_time as time")

	type summary struct {
		bun.BaseModel `bun:"table:trials"`
		Metrics       map[string]any
	}
	var summaryMetrics summary
	if err := db.Bun().NewSelect().Table("trials").
		ColumnExpr("summary_metrics->? AS metrics", model.TrialSummaryMetricsJSONPath(metricGroup)).
		Where("id = ?", trialID).
		Scan(context.TODO(), &summaryMetrics); err != nil {
		return nil, fmt.Errorf("getting summary metrics for trial %d: %w", trialID, err)
	}

	for _, metricName := range append(metricNames, "epoch") {
		metricType := db.MetricTypeString
		if curSummary, ok := summaryMetrics.Metrics[metricName].(map[string]any); ok {
			if m, ok := curSummary["type"].(string); ok {
				metricType = m
			}
		}

		cast := "text"
		switch metricType {
		case db.MetricTypeNumber:
			cast = "float8"
		case db.MetricTypeBool:
			cast = "boolean"
		}
		subq = subq.ColumnExpr("(metrics->?->>?)::? as ?",
			model.TrialMetricsJSONPath(metricGroup == model.ValidationMetricGroup),
			metricName, bun.Safe(cast), bun.Ident(strings.ReplaceAll(metricName, ".", "·")))
	}

	subq = subq.Where("trial_id = ?", trialID).OrderExpr("random()").
		Limit(maxDatapoints)
	switch timeSeriesFilter {
	case nil:
		orderColumn = batches
		subq = subq.Where("total_batches >= ?", startBatches).
			Where("total_batches <= 0 OR total_batches <= ?", endBatches).
			Where("end_time > ?", startTime)
	default:
		orderColumn = strings.ReplaceAll(timeSeriesColumn, ".", "·")
		subq, err = db.ApplyPolymorphicFilter(subq, queryColumn, timeSeriesFilter)
		if err != nil {
			return metricMeasurements, errors.Wrapf(err, "failed to get metrics to sample for experiment")
		}
	}

	metricMeasurements = []db.MetricMeasurements{}
	var results []map[string]interface{}
	err = db.Bun().NewSelect().TableExpr("(?) as downsample", subq).
		OrderExpr(orderColumn).Scan(context.TODO(), &results)
	if err != nil {
		return metricMeasurements, errors.Wrapf(err, "failed to get metrics to sample for experiment")
	}

	selectMetrics := map[string]string{}

	for i := range metricNames {
		selectMetrics[strings.ReplaceAll(metricNames[i], ".", "·")] = metricNames[i]
	}

	for i := range results {
		valuesMap := make(map[string]interface{})
		for mName, mVal := range results[i] {
			if selectMetrics[mName] != "" {
				valuesMap[selectMetrics[mName]] = mVal
			}
		}
		var epoch *float64
		if results[i]["epoch"] != nil {
			if e, ok := results[i]["epoch"].(float64); ok {
				epoch = &e
			} else {
				return nil, fmt.Errorf(
					"metric 'epoch' has nonnumeric value reported value='%v'", results[i]["epoch"])
			}
		}
		var endTime time.Time
		if results[i]["time"] == nil {
			endTime = time.Time{}
		} else {
			endTime = results[i]["time"].(time.Time)
		}
		metricM := db.MetricMeasurements{
			Batches: uint(results[i]["batches"].(int64)),
			Time:    endTime,
			Epoch:   epoch,
			TrialID: int32(results[i]["trial_id"].(int64)),
			Values:  valuesMap,
		}

		metricMeasurements = append(metricMeasurements, metricM)
	}
	return metricMeasurements, nil
}

// CreateTrialSourceInfo creates a TrialSourceInfo object, which allows us to keep
// track of the linkage between an inference/fine tuning trial and its checkpoint/model version.
func CreateTrialSourceInfo(ctx context.Context, tsi *trialv1.TrialSourceInfo,
) (*apiv1.ReportTrialSourceInfoResponse, error) {
	resp := &apiv1.ReportTrialSourceInfoResponse{}
	query := db.Bun().NewInsert().Model(tsi).
		Value("trial_source_info_type", "?", tsi.TrialSourceInfoType.String()).
		Returning("trial_id").Returning("checkpoint_uuid").
		On("CONFLICT (trial_id, checkpoint_uuid) DO UPDATE")
	if tsi.ModelId == nil {
		query.ExcludeColumn("model_id")
	}
	if tsi.ModelVersion == nil {
		query.ExcludeColumn("model_version")
	}
	_, err := query.Exec(ctx, resp)
	return resp, err
}

// Trial is a better bun trial model than the one in pkg/model/experiment.go.
type Trial struct {
	bun.BaseModel         `bun:"table:trials"`
	ID                    int            `bun:"id,pk,autoincrement"`
	ExperimentID          int            `bun:"experiment_id"`
	State                 model.State    `bun:"state"`
	StartTime             time.Time      `bun:"start_time"`
	EndTime               *time.Time     `bun:"end_time"`
	Hparams               map[string]any `bun:"hparams"`
	WarmStartCheckpointID *int           `bun:"warm_start_checkpoint_id"`
	Seed                  int            `bun:"seed"`
	RequestID             *string        `bun:"request_id"`
	BestValidationID      *int           `bun:"best_validation_id"`
	// TODO(ilia): enum for training/validating/checkpointing.
	RunnerState string `bun:"runner_state"`
	RunID       int    `bun:"run_id"`
	Restarts    int    `bun:"restarts"`
	// Note: Tags map values are always "".
	Tags                      map[string]string `bun:"tags"`
	CheckpointSize            int               `bun:"checkpoint_size"`
	CheckpointCount           int               `bun:"checkpoint_count"`
	SearcherMetricValue       *float64          `bun:"searcher_metric_value"`
	SearcherMetricValueSigned *float64          `bun:"searcher_metric_value_signed"`
	TotalBatches              int               `bun:"total_batches"`
	// TODO(ilia): better typing for SummaryMetrics.
	SummaryMetrics          map[string]any `bun:"summary_metrics"`
	SummaryMetricsTimestamp *time.Time     `bun:"summary_metrics_timestamp"`
	LatestValidationID      int            `bun:"latest_validation_id"`
	LastActivity            *time.Time     `bun:"last_activity"`
	ExternalTrialID         *string        `bun:"external_trial_id"`
}

// LatestCheckpointForTrialTx finds the latest completed checkpoint for a trial, returning nil if
// none exists.
func LatestCheckpointForTrialTx(ctx context.Context, idb bun.IDB, trialID int) (
	*model.Checkpoint, error,
) {
	var checkpoint model.Checkpoint
	err := idb.NewSelect().Model(&checkpoint).
		Where("trial_id = ?", trialID).
		Where("state = 'COMPLETED'").
		Order("steps_completed DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, db.MatchSentinelError(err)
	}
	return &checkpoint, nil
}

// UpdateUnmanagedExperimentStatesTx updates an [unmanaged] experiment state according to its
// constituent trial states.
func UpdateUnmanagedExperimentStatesTx(
	ctx context.Context, tx bun.IDB, experiments []*model.Experiment,
) error {
	var trialsRes []Trial
	experimentIDs := make([]int, 0, len(experiments))
	for _, exp := range experiments {
		experimentIDs = append(experimentIDs, exp.ID)
	}

	if err := tx.NewSelect().Model(&trialsRes).
		Column("id", "state", "end_time", "experiment_id").
		Where("experiment_id in (?)", bun.In(experimentIDs)).
		Order("end_time DESC NULLS LAST").
		Scan(ctx); err != nil {
		return err
	}

	if len(trialsRes) == 0 {
		return nil
	}

	groupedTrials := map[int][]Trial{}
	for _, t := range trialsRes {
		groupedTrials[t.ExperimentID] = append(groupedTrials[t.ExperimentID], t)
	}

	// TODO(ilia): rewrite to do it in a single UPDATE query.
	for _, exp := range experiments {
		oldState := exp.State
		trials := groupedTrials[exp.ID]

		if len(trials) == 0 {
			continue
		}

		mostProgressedTrialState := model.PausedState
		trialStateIndex := map[model.State]int{
			model.PausedState:    10,
			model.ErrorState:     20,
			model.CompletedState: 30,
			model.RunningState:   40,
		}

		for _, trial := range trials {
			if newIdx, ok := trialStateIndex[trial.State]; ok {
				if newIdx > trialStateIndex[mostProgressedTrialState] {
					mostProgressedTrialState = trial.State
				}
			}
		}
		exp.State = mostProgressedTrialState

		if exp.State == oldState {
			continue
		}
		columns := []string{"state"}

		if model.TerminalStates[exp.State] {
			columns = append(columns, "end_time")

			var endTime *time.Time

			for _, trial := range trials {
				if trial.EndTime != nil && (endTime == nil || trial.EndTime.After(*endTime)) {
					endTime = trial.EndTime
				}
			}
			if endTime == nil {
				endTime = ptrs.Ptr(time.Now())
			}
			exp.EndTime = endTime
		}

		if _, err := tx.NewUpdate().Model(exp).Column(columns...).WherePK().Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

// MarkLostTrials marks the trials which did not have a heartbeat
// for more than 5 minutes as errored.
func MarkLostTrials(ctx context.Context) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		type ExpWithTrials struct {
			ID      int
			State   model.State
			TrialID int
		}
		var res []ExpWithTrials
		_, err := tx.NewUpdate().Model(&res).
			ModelTableExpr("trials").Table("experiments").
			Where("trials.experiment_id = experiments.id").
			Where("experiments.unmanaged = true").
			Where("trials.state = ?", model.RunningState).
			Where("trials.last_activity < ?", time.Now().Add(-5*time.Minute)).
			Set("state = ?", model.ErrorState).
			Set("end_time = trials.last_activity").
			Returning("experiments.id, experiments.state, trials.id as trial_id").Exec(ctx)

		if err != nil {
			return err
		} else if len(res) == 0 {
			return nil
		}

		experimentsSet := map[int]*model.Experiment{}
		trialIds := []string{}

		for _, r := range res {
			exp := model.Experiment{
				ID:    r.ID,
				State: r.State,
			}
			experimentsSet[exp.ID] = &exp
			trialIds = append(trialIds, strconv.Itoa(r.TrialID))
		}

		log.Infof("marked timed out trials: %s", strings.Join(trialIds, ","))

		experiments := maps.Values(experimentsSet)

		if err := UpdateUnmanagedExperimentStatesTx(ctx, tx, experiments); err != nil {
			return err
		}

		// TODO(ilia): Similarly to `Allocation.sendTaskLog`, write to the updated trial's logs
		// the reason why it switched to the errored state.

		return nil
	})
}

// MarkLostTrialsWorker runs `MarkLostTrials` every 5 minutes.
func MarkLostTrialsWorker(ctx context.Context) {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		err := MarkLostTrials(ctx)
		if err != nil {
			log.Error("error marking timed out unmanaged trials: ", err.Error())
		}

		select {
		case <-t.C:
		case <-ctx.Done():
			return
		}
	}
}

var queryMap db.StaticQueryMap

// ProtoGetTrialsPlusTx does the `proto_get_trials_plus` thing.
func ProtoGetTrialsPlusTx(
	ctx context.Context, idb bun.IDB, trialIDs []int,
) ([]*trialv1.Trial, error) {
	query := queryMap.GetOrLoad("proto_get_trials_plus")

	valuesExpr := make([]string, 0, len(trialIDs))
	trialIDsWithOrdering := make([]any, 0, len(trialIDs))

	for i, trialID := range trialIDs {
		valuesExpr = append(valuesExpr, "(?::int, ?::int)")
		trialIDsWithOrdering = append(trialIDsWithOrdering, trialID, i)
	}

	values := strings.Join(valuesExpr, ", ")
	query = fmt.Sprintf(query, values)

	res := []*trialv1.Trial{}
	resMaps := []map[string]interface{}{}
	if err := db.MatchSentinelError(
		idb.NewRaw(query, trialIDsWithOrdering...).Scan(ctx, &resMaps),
	); err != nil {
		return nil, err
	}

	if len(resMaps) == 0 {
		return nil, db.ErrNotFound
	}

	for _, resMap := range resMaps {
		trial := trialv1.Trial{}
		// Cast string -> []byte `ParseMapToProto` magic.
		jsonFields := []string{
			"hparams", "summary_metrics",
			"best_validation", "latest_validation", "best_checkpoint",
			"task_ids",
		}
		for _, field := range jsonFields {
			switch sVal := resMap[field].(type) {
			case string:
				resMap[field] = []byte(sVal)
			}
		}

		if err := db.ParseMapToProto(resMap, &trial); err != nil {
			return nil, fmt.Errorf("failed to parse map into proto: %w", err)
		}
		res = append(res, &trial)
	}

	return res, nil
}
