package hpimportance

import (
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

type metricType int

// Training designates metrics from training steps.
const Training metricType = 0

// Validation designates metrics from validation steps.
const Validation metricType = 1

// Pending indicates that a computation request is queued.
const Pending string = "pending"

// InProgress indicates that a computation request is in-progress.
const InProgress string = "in_progress"

// Complete indicates that one request was completed and no further requests have been received.
const Complete string = "complete"

// Failed indicates that there was an error during computation.
const Failed string = "failed"

const (
	// RootAddr is the path to use for looking up the manager actor.
	RootAddr = "hpimportance"

	// Evaluate after every 10%, but no more than every 10 minutes
	minPause   = 10 * time.Minute
	minPercent = 0.1
)

// TerminalStates indicate final states, as opposed to tasks that imply imminent changes.
var TerminalStates = map[string]bool{
	Complete: true,
	Failed:   true,
}

// ExperimentCreated is the message an experiment sends when created.
type ExperimentCreated struct {
	ID int
}

// ExperimentCompleted is the message an experiment sends upon completion.
type ExperimentCompleted struct {
	ID int
}

// ExperimentPaused is the message an experiment sends on pausing.
type ExperimentPaused struct {
	ID int
}

// ExperimentProgress is the message an experiment sends after trial completion.
type ExperimentProgress struct {
	ID       int
	Progress float64
}

// The zero-values of these types happen to be sensible defaults too. If we add fields for which
// that is not true, add a newStateRecord(). This is for state that doesn't need to be persisted,
// especially if it is access very frequently.
type stateRecord struct {
	lastResult   time.Time
	lastProgress float64
}

type manager struct {
	db    *db.PgDB
	state map[int]stateRecord
}

// NewManager initializes the master actor (of which there should only be one instance running).
func NewManager(db *db.PgDB) actor.Actor {
	return &manager{db, make(map[int]stateRecord)}
}

func (m *manager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		// TODO: fetch any pending or in_progress tasks from the DB and trigger them
	case actor.PostStop:
		// Do nothing
	case actor.ChildFailed:
		ctx.Log().Warnf("hyperparameter importance worker failed: %+v", msg)
	case actor.ChildStopped:
		// Do nothing - it'll respawn next time a request is received
	case ExperimentCompleted:
		m.triggerDefaultWork(ctx, msg.ID)
	case ExperimentCreated:
		m.state[msg.ID] = stateRecord{
			lastResult: time.Now(),
		}
	case ExperimentPaused:
		m.triggerDefaultWork(ctx, msg.ID)
	case ExperimentProgress:
		var state stateRecord
		var ok bool
		if state, ok = m.state[msg.ID]; !ok {
			state = stateRecord{}
		}
		if msg.Progress-state.lastProgress > minPercent &&
			time.Since(state.lastResult) > minPause {
			m.triggerDefaultWork(ctx, msg.ID)
		}
	case workStarted:
		hpi, err := m.db.GetHPImportance(msg.experimentID)
		if err != nil {
			ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
			return nil
		}
		metricData := getMetricHPImportance(hpi, msg.metricName, msg.metricType)
		metricData.Status = InProgress
		setMetricHPImportance(&hpi, metricData, msg.metricName, msg.metricType)
		err = m.db.SetHPImportance(msg.experimentID, hpi)
		if err != nil {
			ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
			return nil
		}
	case workFailed:
		hpi, err := m.db.GetHPImportance(msg.experimentID)
		if err != nil {
			ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
			return nil
		}
		metricData := getMetricHPImportance(hpi, msg.metricName, msg.metricType)
		metricData.Status = Failed
		metricData.Error = msg.err
		setMetricHPImportance(&hpi, metricData, msg.metricName, msg.metricType)
		err = m.db.SetHPImportance(msg.experimentID, hpi)
		if err != nil {
			ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
			return nil
		}
	case workCompleted:
		m.state[msg.experimentID] = stateRecord{
			lastResult:   time.Now(),
			lastProgress: m.state[msg.experimentID].lastProgress,
		}
		hpi, err := m.db.GetHPImportance(msg.experimentID)
		if err != nil {
			ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
			return nil
		}
		metricData := getMetricHPImportance(hpi, msg.metricName, msg.metricType)
		metricData.ExperimentProgress = msg.progress
		metricData.HpImportance = msg.results
		switch metricData.Status {
		case Pending:
			// Do nothing - this means another startWork message was already sent
		case InProgress:
			metricData.Status = Complete
		default:
			ctx.Log().Warnf("work was completed for a metric with an unexpected state")
		}
		setMetricHPImportance(&hpi, metricData, msg.metricName, msg.metricType)
		err = m.db.SetHPImportance(msg.experimentID, hpi)
		if err != nil {
			ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
			return nil
		}
	default:
		ctx.Log().Errorf("unknown message received by hyperparameter importance manager: %v!",
			ctx.Message())
	}
	return nil
}

func (m *manager) getChild(ctx *actor.Context, experimentID int) *actor.Ref {
	var result *actor.Ref
	/*
		Currently each experiment gets their own actor, that will persist even after the work
		for an experiment may be complete. But the child actor's don't need to maintain any state
		between tasks, so if this becomes a problem in practice we could use a threadpool model
		where the manager maintains a queue of tasks, workers .Ask() the manager for a task (and it
		may return a task to sleep for a time), and the manager decides when to spawn or scale down
		workers. Or each authenticated user can have their own actor, for improved multi-tenancy.
	*/
	result = ctx.Child(experimentID)
	if result == nil {
		w := newWorker(m.db, ctx.Self())
		result, _ = ctx.ActorOf(experimentID, w)
	}
	return result
}

func (m *manager) triggerDefaultWork(ctx *actor.Context, experimentID int) {
	child := m.getChild(ctx, experimentID)

	hpi, err := m.db.GetHPImportance(experimentID)
	if err != nil {
		ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
		return
	}

	config, err := m.db.ExperimentConfig(experimentID)
	if err != nil {
		ctx.Log().Errorf("error retrieving experiment config: %s", err.Error())
		return
	}

	loss := "loss"
	triggerForLoss := false
	lossHpi := getMetricHPImportance(hpi, loss, Training)
	if lossHpi.Status != Pending {
		triggerForLoss = true
	}
	lossHpi.Status = Pending
	setMetricHPImportance(&hpi, lossHpi, loss, Training)

	searcherMetric := config.Searcher.Metric
	triggerForSearcherMetric := false
	searcherMetricHpi := getMetricHPImportance(hpi, searcherMetric, Validation)
	if searcherMetricHpi.Status != Pending {
		triggerForSearcherMetric = true
	}
	searcherMetricHpi.Status = Pending
	setMetricHPImportance(&hpi, searcherMetricHpi, searcherMetric, Validation)

	err = m.db.SetHPImportance(experimentID, hpi)
	if err != nil {
		ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
		return
	}

	if triggerForLoss {
		ctx.Tell(child, startWork{
			experimentID: experimentID,
			metricName:   loss,
			metricType:   Training,
		})
	}
	if triggerForSearcherMetric {
		ctx.Tell(child, startWork{
			experimentID: experimentID,
			metricName:   searcherMetric,
			metricType:   Validation,
		})
	}
}

func setMetricHPImportance(hpi *model.ExperimentHPImportance, metricHpi model.MetricHPImportance,
	metricName string, metricType metricType) *model.ExperimentHPImportance {
	switch metricType {
	case Training:
		hpi.TrainingMetrics[metricName] = metricHpi
	case Validation:
		hpi.ValidationMetrics[metricName] = metricHpi
	default:
		panic("Invalid metric type!")
	}
	return hpi
}

func getMetricHPImportance(hpi model.ExperimentHPImportance, metricName string,
	metricType metricType) model.MetricHPImportance {
	switch metricType {
	case Training:
		metricHpi, ok := hpi.TrainingMetrics[metricName]
		if !ok {
			var newMetricHpi model.MetricHPImportance
			hpi.TrainingMetrics[metricName] = newMetricHpi
			metricHpi = newMetricHpi
		}
		return metricHpi
	case Validation:
		metricHpi, ok := hpi.ValidationMetrics[metricName]
		if !ok {
			var newMetricHpi model.MetricHPImportance
			hpi.ValidationMetrics[metricName] = newMetricHpi
			metricHpi = newMetricHpi
		}
		return metricHpi
	default:
		panic("Invalid metric type!")
	}
}
