package hpimportance

import (
	"fmt"
	"os"
	"path"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Messages used by the HP importance workers.
type (
	startWork struct {
		experimentID int
		metricName   string
		metricType   model.MetricType
		config       HPImportanceConfig
	}

	workStarted struct {
		experimentID int
		metricName   string
		metricType   model.MetricType
	}

	workCompleted struct {
		experimentID int
		metricName   string
		metricType   model.MetricType
		progress     float64
		results      map[string]float64
	}

	workFailed struct {
		experimentID int
		metricName   string
		metricType   model.MetricType
		err          string
	}
)

func taskHandlerFactory(db *db.PgDB, system *actor.System, growforest string, workingDir string,
) func(uint64, interface{}, *actor.Context) interface{} {
	getManager := func() *actor.Ref {
		return system.Get(actor.Addr(RootAddr))
	}

	sendWorkStarted := func(system *actor.System, work startWork) {
		system.Tell(getManager(), workStarted{
			experimentID: work.experimentID,
			metricType:   work.metricType,
			metricName:   work.metricName,
		})
	}

	sendWorkFailed := func(system *actor.System, work startWork, err string) {
		system.Tell(getManager(), workFailed{
			experimentID: work.experimentID,
			metricType:   work.metricType,
			metricName:   work.metricName,
			err:          err,
		})
	}

	sendWorkCompleted := func(system *actor.System, work startWork, progress float64,
		results map[string]float64) {
		system.Tell(getManager(), workCompleted{
			experimentID: work.experimentID,
			metricType:   work.metricType,
			metricName:   work.metricName,
			progress:     progress,
			results:      results,
		})
	}

	return func(actorId uint64, task interface{}, ctx *actor.Context) interface{} {
		work, ok := task.(startWork)
		if !ok {
			panic("invalid task passed to hp importance actor pool")
		}
		sendWorkStarted(system, work)

		state, progress, err := db.GetExperimentStatus(work.experimentID)
		if err != nil {
			sendWorkFailed(system, work, err.Error())
			return nil
		}
		if state == model.CompletedState {
			progress = 1
		}

		masterConfig := work.config

		experimentConfig, err := db.ExperimentConfig(work.experimentID)
		if err != nil {
			sendWorkFailed(system, work, err.Error())
			return nil
		}

		var trials map[int][]model.HPImportanceTrialData
		switch work.metricType {
		case model.TrainingMetric:
			trials, err = db.FetchHPImportanceTrainingData(work.experimentID, work.metricName)
			if err != nil {
				sendWorkFailed(system, work, err.Error())
				return nil
			}
		case model.ValidationMetric:
			trials, err = db.FetchHPImportanceValidationData(work.experimentID, work.metricName)
			if err != nil {
				sendWorkFailed(system, work, err.Error())
				return nil
			}
		default:
			sendWorkFailed(system, work, "invalid metric type received in hyperparameter importance worker")
			return nil
		}
		taskDir := path.Join(workingDir, fmt.Sprint(actorId))
		err = os.Mkdir(taskDir, 0066)
		if err != nil {
			sendWorkFailed(system, work, err.Error())
		}
		results := computeHPImportance(trials, experimentConfig, masterConfig, growforest, taskDir)
		err = os.RemoveAll(taskDir)
		if err != nil {
			ctx.Log().Errorf("Failed to clean up temporary directory %s", taskDir)
		}
		sendWorkCompleted(system, work, progress, results)
		return nil
	}
}
