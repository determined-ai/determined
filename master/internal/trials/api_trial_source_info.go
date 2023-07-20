package trials

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// TrialSourceInfoAPIServer is a dummy struct to do dependency injection to the api.
// This allows us to define apiServer functions in sub-modules.
type TrialSourceInfoAPIServer struct{}

// ReportTrialSourceInfo creates a TrialSourceInfo, which serves as a link between
// trials and checkpoints used for tracking purposes for fine tuning and inference.
func (a *TrialSourceInfoAPIServer) ReportTrialSourceInfo(
	ctx context.Context, req *apiv1.ReportTrialSourceInfoRequest,
) (*apiv1.ReportTrialSourceInfoResponse, error) {
	tsi := req.TrialSourceInfo
	if err := CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(tsi.TrialId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	resp := &apiv1.ReportTrialSourceInfoResponse{}
	query := db.Bun().NewInsert().Model(tsi).
		Value("trial_source_info_type", "?", tsi.TrialSourceInfoType.String()).
		Returning("trial_id").Returning("checkpoint_uuid")
	if tsi.ModelVersionId == nil {
		query.ExcludeColumn("model_version_id")
	}
	if tsi.ModelVersionVersion == nil {
		query.ExcludeColumn("model_version_version")
	}
	_, err := query.Exec(ctx, resp)
	return resp, err
}

// GetMetricsForTrialSourceInfoQuery takes in a bun.SelectQuery on the
// trial_source_infos table, and fetches the metrics for each of the connected trials.
func GetMetricsForTrialSourceInfoQuery(
	ctx context.Context, q *bun.SelectQuery,
) ([]*apiv1.TrialSourceInfoMetric, error) {
	trialIds := []struct {
		TrialID             int
		TrialSourceInfoType string
	}{}
	q = q.Column("trial_id", "trial_source_info_type")
	err := q.Scan(ctx, &trialIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get trial source info %w", err)
	}

	ret := []*apiv1.TrialSourceInfoMetric{}
	numMetricsLimit := 1000
	for _, val := range trialIds {
		if err := CanGetTrialsExperimentAndCheckCanDoAction(ctx, val.TrialID,
			experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
			// If the user can see the checkpoint, but not one of the inference
			// or fine tuning trials that points to it, simply don't show those
			// particular trials.
			continue
		}
		sourceType := trialv1.TrialSourceInfoType_value[val.TrialSourceInfoType]
		res, err := db.GetMetrics(ctx, val.TrialID, -1, numMetricsLimit, model.InferenceMetricGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to get metrics %w", err)
		}
		trialSourceInfoMetric := &apiv1.TrialSourceInfoMetric{
			TrialId:             int32(val.TrialID),
			TrialSourceInfoType: trialv1.TrialSourceInfoType(sourceType),
			MetricReports:       res,
		}
		ret = append(ret, trialSourceInfoMetric)
	}
	return ret, nil
}
