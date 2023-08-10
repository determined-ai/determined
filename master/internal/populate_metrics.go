package internal

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc/metadata"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/sproto"
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

const trainingToValidationRatio = 10

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

func reportNonTrivialMetrics(ctx context.Context, api *apiServer, trialID int32,
	batches int,
) error {
	fmt.Println("non trivial metrics for", batches, "batches") //nolint:forbidigo
	N := 5
	losses := []float64{}
	for i := 0; i < N; i++ {
		losses = append(losses, rand.Float64()) //nolint: gosec
	}

	type Factor struct {
		a, b float64
	}

	factors := []Factor{}
	for i := 0; i < N; i++ {
		factors = append(factors, Factor{rand.Float64(), rand.Float64() / 10}) //nolint: gosec
	}

	printTime := 200
	start := time.Now()
	for b := 0; b < batches; b++ {
		if b%printTime == 0 {
			start = time.Now()
		}
		for i := 0; i < N; i++ {
			val := float64(1)
			if rand.Float64() <= factors[i].a { //nolint: gosec
				val = float64(-1)
			}
			losses[i] *= (1 - (val * rand.Float64() * factors[i].b)) //nolint: gosec
		}
		trainingAvgMetrics := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"loss": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: losses[0], //nolint: gosec
					},
				},
				"loss2": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: losses[1], //nolint: gosec
					},
				},
				"loss3": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: losses[2], //nolint: gosec
					},
				},
			},
		}

		trainingMetrics := trialv1.TrialMetrics{
			TrialId:        trialID,
			StepsCompleted: int32(b + 1),
			Metrics: &commonv1.Metrics{
				AvgMetrics: trainingAvgMetrics,
			},
		}

		err := db.SingleDB().AddTrainingMetrics(ctx, &trainingMetrics)
		if err != nil {
			return err
		}

		validationAvgMetrics := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"loss": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: losses[0], //nolint: gosec
					},
				},
				"loss2": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: losses[3], //nolint: gosec
					},
				},
				"loss3": {
					Kind: &structpb.Value_NumberValue{
						NumberValue: losses[4], //nolint: gosec
					},
				},
			},
		}

		validationMetrics := trialv1.TrialMetrics{
			TrialId:        trialID,
			StepsCompleted: int32(b + 1),
			Metrics: &commonv1.Metrics{
				AvgMetrics: validationAvgMetrics,
			},
		}
		if b%trainingToValidationRatio == 0 {
			err = db.SingleDB().AddValidationMetrics(ctx, &validationMetrics)
		}

		if err != nil {
			return err
		}
		if b%printTime == 0 {
			fmt.Println("batch time after these many batches", time.Since(start), b) //nolint:forbidigo
		}
	}

	return nil
}

func reportTrivialMetrics(ctx context.Context, api *apiServer, trialID int32, batches int) error {
	fmt.Println("trivial metrics for", batches, "batches") //nolint:forbidigo

	trainingMetrics := trialv1.TrialMetrics{
		TrialId:        trialID,
		StepsCompleted: int32(batches),
		Metrics: &commonv1.Metrics{
			AvgMetrics: makeMetrics(),
		},
	}

	err := db.SingleDB().AddTrainingMetrics(ctx, &trainingMetrics)
	if err != nil {
		return err
	}

	validationMetrics := trialv1.TrialMetrics{
		TrialId:        trialID,
		StepsCompleted: int32(batches),
		Metrics: &commonv1.Metrics{
			AvgMetrics: makeMetrics(),
		},
	}

	if batches%trainingToValidationRatio == 0 {
		err = db.SingleDB().AddValidationMetrics(ctx, &validationMetrics)
	}

	if err != nil {
		return err
	}

	return nil
}

// PopulateExpTrialsMetrics adds metrics for a trial and exp to db.
func PopulateExpTrialsMetrics(pgdb *db.PgDB, masterConfig *config.Config, trivialMetrics bool,
	batches int,
) error {
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
	jID := model.NewJobID()
	exp := &model.Experiment{
		JobID:                jID,
		State:                model.CompletedState,
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
	// create task
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

	tr := model.Trial{
		ExperimentID: exp.ID,
		State:        model.CompletedState,
		StartTime:    time.Now(),
	}
	if err = db.AddTrial(ctx, &tr, tID); err != nil {
		return err
	}
	if trivialMetrics {
		return reportTrivialMetrics(ctx, api, int32(tr.ID), batches)
	}
	return reportNonTrivialMetrics(ctx, api, int32(tr.ID), batches)
}
