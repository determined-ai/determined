package internal

import (
	"context"
	"fmt"
	"math/rand"
	"time"

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
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
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

func waitForExperimentToComplete(api *apiServer, ctx context.Context, expId int32, maxWaitSecs int) (bool, experimentv1.State) {
	targetState := experimentv1.State_STATE_COMPLETED
	var expState experimentv1.State
	for i := 0; i < maxWaitSecs; i++ {
		expReq := apiv1.GetExperimentRequest{
			ExperimentId: expId,
		}
		resp, err := api.GetExperiment(ctx, &expReq)
		log.Debugf("Get experiment: %v", err)
		expState = resp.Experiment.State
		if expState == targetState {
			return true, expState
		} else if expState == experimentv1.State_STATE_CANCELED ||
			expState == experimentv1.State_STATE_ERROR {
			return false, expState
		}
	}

	return false, expState
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
	activeConfig := expconf.ExperimentConfig{
		RawSearcher: &expconf.SearcherConfig{
			RawMetric: ptrs.Ptr("loss"),
			RawSingleConfig: &expconf.SingleConfig{
				RawMaxLength: &maxLength,
			},
		},
		RawEntrypoint:      &expconf.Entrypoint{RawEntrypoint: "model_def:SomeTrialClass"},
		RawHyperparameters: expconf.Hyperparameters{},
		RawCheckpointStorage: &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfig{
				RawHostPath: ptrs.Ptr("/"),
			},
		},
		RawMaxRestarts: &maxRestarts,
	}
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
	err = pgdb.AddTask(tIn)

	// create trial

	tr := model.Trial{
		TaskID:       tID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	err = pgdb.AddTrial(&tr)

	return reportMetrics(api, ctx, int32(tr.ID)) // single searcher so there's only one trial
}
