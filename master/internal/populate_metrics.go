package internal

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/metadata"
)

func makeMetrics() *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"loss1": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: rand.Float64(), //nolint: gosec
				},
			},
			"loss2": {
				Kind: &structpb.Value_NumberValue{
					NumberValue: rand.Float64(), //nolint: gosec
				},
			},
		},
	}
}

func reportMetrics(api *apiServer, ctx context.Context, trialID int32) error {

	trainingbBatchMetrics := []*structpb.Struct{}
	const stepSize = 500
	for j := 0; j < stepSize; j++ {
		trainingbBatchMetrics = append(trainingbBatchMetrics, makeMetrics())
	}

	trainingMetrics := trialv1.TrialMetrics{
		TrialId:        trialID,
		StepsCompleted: stepSize,
		Metrics: &commonv1.Metrics{
			AvgMetrics:   makeMetrics(),
			BatchMetrics: trainingbBatchMetrics,
		},
	}

	_, err := api.ReportTrialTrainingMetrics(ctx,
		&apiv1.ReportTrialTrainingMetricsRequest{
			TrainingMetrics: &trainingMetrics,
		})

	if err != nil {
		return err
	}

	validationBatchMetrics := []*structpb.Struct{}

	for j := 0; j < stepSize; j++ {
		validationBatchMetrics = append(validationBatchMetrics, makeMetrics())
	}

	validationMetrics := trialv1.TrialMetrics{
		TrialId:        trialID,
		StepsCompleted: stepSize,
		Metrics: &commonv1.Metrics{
			AvgMetrics:   makeMetrics(),
			BatchMetrics: trainingbBatchMetrics,
		},
	}

	_, err = api.ReportTrialValidationMetrics(ctx,
		&apiv1.ReportTrialValidationMetricsRequest{
			ValidationMetrics: &validationMetrics,
		})

	if err != nil {
		return err
	}

	return nil
}

func PopulateExpTrialsMetrics(pgdb *db.PgDB, masterConfig *config.Config) error {
	system := actor.NewSystem("mock")
	ref, _ := system.ActorOf(sproto.AgentRMAddr, actor.ActorFunc(
		func(context *actor.Context) error {
			switch context.Message().(type) {
			case sproto.DeleteJob:
				context.Respond(sproto.EmptyDeleteJobResponse())
			}
			return nil
		}))
	mockRM := actorrm.Wrap(ref)
	api := &apiServer{
		m: &Master{
			trialLogBackend: pgdb,
			system:          system,
			db:              pgdb,
			taskLogBackend:  pgdb,
			rm:              mockRM,
			config:          masterConfig,
			taskSpec:        &tasks.TaskSpec{},
		},
	}

	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: "admin"})
	if err != nil {
		return err
	}

	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))

	conf := `
	searcher:
		name: single
		metric: x
		max_length: 1

	max_restarts: 0`
	createReq := &apiv1.CreateExperimentRequest{
		ModelDefinition: []*utilv1.File{{Content: []byte{1}}},
		Config:          conf,
		ParentId:        0,
		Activate:        false,
		ProjectId:       1,
	}

	resp2, err := api.CreateExperiment(ctx, createReq)
	if err != nil {
		return err
	}

	getTrialsReq := &apiv1.GetExperimentTrialsRequest{
		ExperimentId: resp2.Experiment.Id,
	}

	resp3, err := api.GetExperimentTrials(ctx, getTrialsReq)
	trials := resp3.Trials
	if len(trials) < 1 {
		return fmt.Errorf("no trials in experiment")
	}

	reportMetrics(api, ctx, trials[0].Id) // single searcher so there's only one trial

}
