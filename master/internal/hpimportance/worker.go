package hpimportance

import (
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

type startWork struct {
	experimentID int
	metricName   string
	metricType   metricType
}

type workStarted struct {
	experimentID int
	metricName   string
	metricType   metricType
}

type workCompleted struct {
	experimentID int
	metricName   string
	metricType   metricType
	progress     float64
	results      map[string]float64
}

type workFailed struct {
	experimentID int
	metricName   string
	metricType   metricType
	err          string
}

type worker struct {
	db      *db.PgDB
	manager *actor.Ref
}

func newWorker(db *db.PgDB, manager *actor.Ref) actor.Actor {
	return &worker{db: db, manager: manager}
}

func (w *worker) sendWorkStarted(ctx *actor.Context, msg startWork) {
	ctx.Tell(w.manager, workStarted(msg))
}

func (w *worker) sendWorkFailed(ctx *actor.Context, msg startWork, err string) {
	ctx.Tell(w.manager, workFailed{
		experimentID: msg.experimentID,
		metricType:   msg.metricType,
		metricName:   msg.metricName,
		err:          err,
	})
}

func (w *worker) sendWorkCompleted(ctx *actor.Context, msg startWork, progress float64,
	results map[string]float64) {
	ctx.Tell(w.manager, workCompleted{
		experimentID: msg.experimentID,
		metricType:   msg.metricType,
		metricName:   msg.metricName,
		progress:     progress,
		results:      results,
	})
}

func (w *worker) Receive(ctx *actor.Context) (err error) {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		// Do nothing
	case actor.PostStop:
		// Do nothing
	case startWork:
		w.sendWorkStarted(ctx, msg)

		state, progress, err := w.db.GetExperimentStatus(msg.experimentID)
		if err != nil {
			w.sendWorkFailed(ctx, msg, err.Error())
		}
		if state == model.CompletedState {
			progress = 1
		}

		var trials *[]model.HPImportanceTrialData
		switch msg.metricType {
		case Training:
			trials, err = w.db.FetchHPImportanceTrainingData(msg.experimentID, msg.metricName)
			if err != nil {
				w.sendWorkFailed(ctx, msg, err.Error())
				return nil
			}
		case Validation:
			trials, err = w.db.FetchHPImportanceValidationData(msg.experimentID, msg.metricName)
			if err != nil {
				w.sendWorkFailed(ctx, msg, err.Error())
				return nil
			}
		default:
			w.sendWorkFailed(ctx, msg, "invalid metric type received in hyperparameter importance worker")
			return nil
		}
		results := computeHPImportance(trials)
		w.sendWorkCompleted(ctx, msg, progress, results)
	default:
		ctx.Log().Errorf("Unknown message sent to HP Importance worker: %v!", ctx.Message())
	}
	return nil
}
