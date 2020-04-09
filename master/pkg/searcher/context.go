package searcher

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

type Sampler func(ctx Context) hparamSample

func RandomSampler(ctx Context) hparamSample {
	return sampleAll(ctx.Hyperparameters(), ctx.Rand())
}

func PreSampled(sample hparamSample) Sampler {
	return func(Context) hparamSample {
		return sample
	}
}

type Context interface {
	Rand() *nprand.State
	Hyperparameters() model.Hyperparameters
	Sample(trial RequestID) hparamSample

	NewTrial(Sampler) RequestID
	NewTrialFromCheckpoint(Sampler, RequestID) RequestID
	TrainAndValidate(trial RequestID, steps int)
	CloseTrial(trial RequestID)
}

type context struct {
	searcher *Searcher
	ops      []Operation
}

func (c *context) Rand() *nprand.State {
	return c.searcher.rand
}

func (c *context) Hyperparameters() model.Hyperparameters {
	return c.searcher.hparams
}

func (c *context) Sample(trial RequestID) hparamSample {
	return c.searcher.samples[trial]
}

func (c *context) NewTrial(sampler Sampler) RequestID {
	create := NewCreate(c.Rand(), sampler(c))
	c.searcher.samples[create.RequestID] = create.Hparams
	c.ops = append(c.ops, create)
	return create.RequestID
}

func (c *context) NewTrialFromCheckpoint(sampler Sampler, trial RequestID) RequestID {
	create := NewCreate(c.Rand(), sampler(c))
	c.searcher.samples[create.RequestID] = create.Hparams
	step := c.searcher.steps[trial]
	if c.searcher.checkpoints[trial] < step {
		checkpoint := WorkloadOperation{
			RequestID: trial,
			Kind:      CheckpointModel,
			StepID:    step,
		}
		c.searcher.pendingCheckpoints[checkpoint] = create
		c.searcher.pendingTrials[create.RequestID] = make([]Operation, 0)
		c.ops = append(c.ops, checkpoint)
	} else {
		c.ops = append(c.ops, create)
	}
	return create.RequestID
}

func (c *context) TrainAndValidate(trial RequestID, steps int) {
	var ops []Operation
	current := c.searcher.steps[trial] + 1
	for end := current + steps; current < end; current++ {
		ops = append(ops, WorkloadOperation{RequestID: trial, Kind: RunStep, StepID: current})
	}
	current--
	ops = append(ops, WorkloadOperation{
		RequestID: trial, Kind: ComputeValidationMetrics, StepID: current})
	c.searcher.steps[trial] = current

	if trialOps, ok := c.searcher.pendingTrials[trial]; ok {
		c.searcher.pendingTrials[trial] = append(trialOps, ops...)
	} else {
		c.ops = append(c.ops, ops...)
	}
}

func (c *context) CloseTrial(trial RequestID) {
	c.ops = append(c.ops, Close{RequestID: trial})
}

func (c *context) pendingOperations() []Operation {
	return c.ops
}
