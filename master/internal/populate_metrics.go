package internal

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/metadata"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
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

func reportMetrics(ctx context.Context, api *apiServer, trialID int32) error {
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
			BatchMetrics: validationBatchMetrics,
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

// PopulateExpTrialsMetrics adds metrics for a trial and exp to db.
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

	_, err := user.UserByUsername("admin")
	if err != nil {
		return err
	}

	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: "admin"})
	if err != nil {
		return err
	}

	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))

	// create exp and config
	maxLength := expconf.NewLengthInBatches(100)
	maxRestarts := 0
	activeConfig := expconf.ExperimentConfig{ //nolint:exhaustivestruct
		RawSearcher: &expconf.SearcherConfig{ //nolint:exhaustivestruct
			RawMetric: ptrs.Ptr("loss"),
			RawSingleConfig: &expconf.SingleConfig{ //nolint:exhaustivestruct
				RawMaxLength: &maxLength,
			},
		},
		RawEntrypoint:      &expconf.Entrypoint{RawEntrypoint: "model_def:SomeTrialClass"},
		RawHyperparameters: expconf.Hyperparameters{},
		RawCheckpointStorage: &expconf.CheckpointStorageConfig{ //nolint:exhaustivestruct
			RawSharedFSConfig: &expconf.SharedFSConfig{ //nolint:exhaustivestruct
				RawHostPath: ptrs.Ptr("/"),
			},
		},
		RawMaxRestarts: &maxRestarts,
	} //nolint:exhaustivestruct
	activeConfig = schemas.WithDefaults(activeConfig)
	model.DefaultTaskContainerDefaults().MergeIntoExpConfig(&activeConfig)

	var defaultDeterminedUID model.UserID = 2
	exp := &model.Experiment{
		JobID:                model.NewJobID(),
		State:                model.ActiveState,
		Config:               activeConfig.AsLegacy(),
		StartTime:            time.Now(),
		OwnerID:              &defaultDeterminedUID,
		ModelDefinitionBytes: []byte{},
		ProjectID:            1,
	}
	err = pgdb.AddExperiment(exp, activeConfig)
	if err != nil {
		return err
	}
	// create job and task
	jID := model.NewJobID()
	jIn := &model.Job{
		JobID:   jID,
		JobType: model.JobTypeExperiment,
		OwnerID: exp.OwnerID,
		QPos:    decimal.New(0, 0),
	}
	err = pgdb.AddJob(jIn)
	if err != nil {
		return err
	}
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:    tID,
		JobID:     &jID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	if err = pgdb.AddTask(tIn); err != nil {
		return err
	}

	// create trial

	tr := model.Trial{
		TaskID:       tID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	if err = pgdb.AddTrial(&tr); err != nil {
		return err
	}

	return reportMetrics(ctx, api, int32(tr.ID)) // single searcher so there's only one trial
}
