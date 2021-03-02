package hpimportance

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/pool"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// RootAddr is the path to use for looking up the manager actor.
	RootAddr = "hpimportance"

	// Evaluate after every 10%, but no more than every 10 minutes
	minPause   = 10 * time.Minute
	minPercent = 0.1

	// Each worker will create a subdirectory of this for CloudForest's input & output
	workingDir = "/tmp/determined/growforest"

	// The name of the CloudForest executable to look for
	growforestBin = "growforest"
)

// HPImportanceConfig is the configuration in the master for hyperparameter importance.
type HPImportanceConfig struct {
	WorkersLimit   uint `json:"workers_limit"`
	QueueLimit     uint `json:"queue_limit"`
	CoresPerWorker uint `json:"cores_per_worker"`
	MaxTrees       uint `json:"max_trees"`
}

// Messages handled by the HP importance manager.
type (
	// ExperimentCreated is the message an experiment sends when created.
	ExperimentCreated struct {
		ID int
	}

	// ExperimentCompleted is the message an experiment sends upon completion.
	ExperimentCompleted struct {
		ID int
	}

	// ExperimentProgress is the message an experiment sends after trial completion.
	ExperimentProgress struct {
		ID       int
		Progress float64
	}

	// WorkRequest is an explicit request to compute HP importance on-demand.
	WorkRequest struct {
		ExperimentID int
		MetricName   string
		MetricType   model.MetricType
	}
)

// The zero-values of these types happen to be sensible defaults too. If we add fields for which
// that is not true, add a newStateRecord(). This is for state that doesn't need to be persisted,
// especially if it is access very frequently.
type stateRecord struct {
	lastResult   time.Time
	lastProgress float64
}

type manager struct {
	config   HPImportanceConfig
	db       *db.PgDB
	state    map[int]stateRecord
	pool     pool.ActorPool
	disabled bool
}

// NewManager initializes the master actor (of which there should only be one instance running).
func NewManager(db *db.PgDB, system *actor.System, config HPImportanceConfig, masterRoot string,
) (actor.Actor, error) {
	// growforest should either be installed in PATH (when running from source) or package with the
	// master (when running from binary packages).
	growforest := path.Join(masterRoot, growforestBin)
	_, err := os.Stat(growforest)
	if os.IsNotExist(err) {
		resolvedPath, err := exec.LookPath(growforestBin)
		if err != nil {
			return nil, fmt.Errorf("Failed to find 'growforest' binary. Install it with " +
				"'go install github.com/ryanbressler/CloudForest/growforest'")
		}
		growforest = resolvedPath
	}

	err = os.MkdirAll(workingDir, 0700)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create scratch space for HP importance computation")
	}

	return &manager{
		config:   config,
		db:       db,
		disabled: config.WorkersLimit == 0,
		state:    make(map[int]stateRecord),
		pool: pool.NewActorPool(
			system, config.QueueLimit, config.WorkersLimit, "hp-importance-pool",
			taskHandlerFactory(db, system, growforest, workingDir), nil,
		),
	}, nil
}

func (m *manager) Receive(ctx *actor.Context) error {
	if m.disabled {
		return nil
	}
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		m.triggerPartialWork(ctx)
	case actor.PostStop:
		// Do nothing
	case actor.ChildFailed:
		ctx.Log().Warnf("hyperparameter importance worker failed: %+v", msg)
	case actor.ChildStopped:
		// Do nothing - it'll respawn next time a request is received
	case ExperimentCompleted:
		m.experimentCompleted(ctx, msg)
	case ExperimentCreated:
		m.experimentCreated(ctx, msg)
	case ExperimentProgress:
		m.experimentProgress(ctx, msg)
	case WorkRequest:
		m.workRequest(ctx, msg)
	case workStarted:
		m.workStarted(ctx, msg)
	case workFailed:
		m.workFailed(ctx, msg)
	case workCompleted:
		m.workCompleted(ctx, msg)
	default:
		ctx.Log().Errorf("unknown message received by hyperparameter importance manager: %v!",
			ctx.Message())
	}
	return nil
}

func (m *manager) triggerPartialWork(ctx *actor.Context) {
	ids, hpis, err := m.db.GetPartialHPImportance()
	if err != nil {
		ctx.Log().Errorf("failed to resume hyperparameter importance work on restart: %v", err)
	}
	for i := 0; i < len(ids) && i < len(hpis); i++ {
		for metric, metricHpi := range hpis[i].TrainingMetrics {
			if metricHpi.Pending || metricHpi.InProgress {
				ctx.Tell(ctx.Self(), startWork{
					experimentID: ids[i],
					metricName:   metric,
					metricType:   model.TrainingMetric,
					config:       m.config,
				})
			}
		}
		for metric, metricHpi := range hpis[i].ValidationMetrics {
			if metricHpi.Pending || metricHpi.InProgress {
				ctx.Tell(ctx.Self(), startWork{
					experimentID: ids[i],
					metricName:   metric,
					metricType:   model.ValidationMetric,
					config:       m.config,
				})
			}
		}
	}
}

func (m *manager) experimentCompleted(ctx *actor.Context, msg ExperimentCompleted) {
	m.triggerDefaultWork(ctx, msg.ID)
}

func (m *manager) experimentCreated(ctx *actor.Context, msg ExperimentCreated) {
	m.state[msg.ID] = stateRecord{
		lastResult: time.Now(),
	}
}

func (m *manager) experimentProgress(ctx *actor.Context, msg ExperimentProgress) {
	state, ok := m.state[msg.ID]
	if !ok {
		state = stateRecord{}
	}
	if msg.Progress-state.lastProgress > minPercent &&
		time.Since(state.lastResult) > minPause {
		m.triggerDefaultWork(ctx, msg.ID)
	}
}

func (m *manager) workStarted(ctx *actor.Context, msg workStarted) {
	hpi, err := m.db.GetHPImportance(msg.experimentID)
	if err != nil {
		ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
		return
	}
	metricData := hpi.GetMetricHPImportance(msg.metricName, msg.metricType)
	metricData.Pending = false
	metricData.InProgress = true
	hpi.SetMetricHPImportance(metricData, msg.metricName, msg.metricType)
	err = m.db.SetHPImportance(msg.experimentID, hpi)
	if err != nil {
		ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
		return
	}
}

func (m *manager) workFailed(ctx *actor.Context, msg workFailed) {
	hpi, err := m.db.GetHPImportance(msg.experimentID)
	if err != nil {
		ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
		return
	}
	metricData := hpi.GetMetricHPImportance(msg.metricName, msg.metricType)
	metricData.InProgress = false
	metricData.Error = msg.err
	hpi.SetMetricHPImportance(metricData, msg.metricName, msg.metricType)

	err = m.db.SetHPImportance(msg.experimentID, hpi)
	if err != nil {
		ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
		return
	}
}

func (m *manager) workCompleted(ctx *actor.Context, msg workCompleted) {
	m.state[msg.experimentID] = stateRecord{
		lastResult:   time.Now(),
		lastProgress: m.state[msg.experimentID].lastProgress,
	}
	hpi, err := m.db.GetHPImportance(msg.experimentID)
	if err != nil {
		ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
		return
	}
	metricData := hpi.GetMetricHPImportance(msg.metricName, msg.metricType)
	metricData.ExperimentProgress = msg.progress
	metricData.HpImportance = msg.results
	metricData.InProgress = false
	hpi.SetMetricHPImportance(metricData, msg.metricName, msg.metricType)
	err = m.db.SetHPImportance(msg.experimentID, hpi)
	if err != nil {
		ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
		return
	}
}

func (m *manager) workRequest(ctx *actor.Context, msg WorkRequest) {
	hpi, err := m.db.GetHPImportance(msg.ExperimentID)
	if err != nil {
		ctx.Log().Errorf("error retrieving hyperparameter importance state: %s", err.Error())
		return
	}

	metricHpi := hpi.GetMetricHPImportance(msg.MetricName, msg.MetricType)
	if metricHpi.Pending {
		return
	}
	metricHpi.Pending = true

	err = m.pool.SubmitTask(startWork{
		experimentID: msg.ExperimentID,
		metricName:   msg.MetricName,
		metricType:   msg.MetricType,
		config:       m.config,
	})
	if err != nil {
		metricHpi.Pending = false
		metricHpi.Error = err.Error()
	}

	hpi.SetMetricHPImportance(metricHpi, msg.MetricName, msg.MetricType)
	err = m.db.SetHPImportance(msg.ExperimentID, hpi)
	if err != nil {
		ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
		return
	}
}

func (m *manager) triggerDefaultWork(ctx *actor.Context, experimentID int) {
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
	lossHpi := hpi.GetMetricHPImportance(loss, model.TrainingMetric)
	if !lossHpi.Pending {
		triggerForLoss = true
		lossHpi.Pending = true
		hpi.SetMetricHPImportance(lossHpi, loss, model.TrainingMetric)
	}

	searcherMetric := config.Searcher.Metric
	triggerForSearcherMetric := false
	searcherMetricHpi := hpi.GetMetricHPImportance(searcherMetric, model.ValidationMetric)
	if !searcherMetricHpi.Pending {
		triggerForSearcherMetric = true
		searcherMetricHpi.Pending = true
		hpi.SetMetricHPImportance(searcherMetricHpi, searcherMetric, model.ValidationMetric)
	}

	if triggerForLoss || triggerForSearcherMetric {
		if triggerForLoss {
			err = m.pool.SubmitTask(startWork{
				experimentID: experimentID,
				metricName:   loss,
				metricType:   model.TrainingMetric,
				config:       m.config,
			})
			if err != nil {
				lossHpi.Pending = false
				lossHpi.Error = err.Error()
				hpi.SetMetricHPImportance(lossHpi, loss, model.TrainingMetric)
			}
		}
		if triggerForSearcherMetric {
			err = m.pool.SubmitTask(startWork{
				experimentID: experimentID,
				metricName:   searcherMetric,
				metricType:   model.ValidationMetric,
				config:       m.config,
			})
			if err != nil {
				searcherMetricHpi.Pending = false
				searcherMetricHpi.Error = err.Error()
				hpi.SetMetricHPImportance(searcherMetricHpi, searcherMetric, model.ValidationMetric)
			}
		}
		err = m.db.SetHPImportance(experimentID, hpi)
		if err != nil {
			ctx.Log().Errorf("error writing hyperparameter importance state: %s", err.Error())
			return
		}
	}
}
