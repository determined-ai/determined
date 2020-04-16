package searcher

import (
	"fmt"

	"github.com/d4l3k/go-bayesopt"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

// bayesSearch implements a bayesian optimization hyperparameter search.
// See https://en.wikipedia.org/wiki/Hyperparameter_optimization#Bayesian_optimization for more
// information.
type bayesSearch struct {
	defaultSearchMethod
	model.BayesConfig
	optimizer     *bayesopt.Optimizer
	trialsStarted int
	params        map[string]bayesParam
	trialSamples  map[RequestID]hparamSample
}

func newBayesSearch(config model.BayesConfig) SearchMethod {
	return &bayesSearch{
		BayesConfig:  config,
		params:       make(map[string]bayesParam),
		trialSamples: make(map[RequestID]hparamSample),
	}
}

func (s *bayesSearch) initialOperations(ctx context) ([]Operation, error) {
	var params []bayesopt.Param
	for name, param := range ctx.hparams {
		p := bayesParam{name: name, Hyperparameter: param, rand: ctx.rand}
		s.params[name] = p
		params = append(params, p)
	}
	s.optimizer = bayesopt.New(params,
		bayesopt.WithMinimize(s.SmallerIsBetter),
		bayesopt.WithExploration(bayesopt.UCB{Kappa: s.Kappa}),
		bayesopt.WithRandomRounds(s.ConcurrentTrials),
		bayesopt.WithRounds(s.MaxTrials))
	var operations []Operation
	for i := 0; i < s.ConcurrentTrials; i++ {
		ops, err := s.newTrial(ctx)
		// This method should not error out on the initial random trial sampling.
		check.Panic(err)
		operations = append(operations, ops...)
	}
	return operations, nil
}

func (s *bayesSearch) validationCompleted(
	ctx context, requestID RequestID, _ Workload, metrics ValidationMetrics,
) ([]Operation, error) {
	if s.trialsStarted >= s.MaxTrials {
		return nil, nil
	}
	value, err := metrics.Metric(s.Metric)
	if err != nil {
		return nil, err
	}
	params := make(map[bayesopt.Param]float64)
	for key, sampleValue := range s.trialSamples[requestID] {
		param := s.params[key]
		params[param] = param.ToFloat(sampleValue)
	}
	s.optimizer.Log(params, value)
	return s.newTrial(ctx)
}

func (s *bayesSearch) newTrial(ctx context) ([]Operation, error) {
	bayesSample, _, err := s.optimizer.Next()
	if err != nil {
		return nil, err
	}
	s.trialsStarted++
	var ops []Operation
	sample := make(hparamSample)
	for param, value := range bayesSample {
		sample[param.GetName()] = s.params[param.GetName()].FromFloat(value)
	}
	create := NewCreate(ctx.rand, sample, model.TrialWorkloadSequencerType)
	s.trialSamples[create.RequestID] = sample
	ops = append(ops, create)
	ops = append(ops, trainAndValidate(create.RequestID, 0, s.MaxSteps)...)
	ops = append(ops, NewClose(create.RequestID))
	return ops, nil
}

func (s *bayesSearch) progress(workloadsCompleted int) float64 {
	return float64(workloadsCompleted) / float64((s.MaxSteps+1)*s.MaxTrials)
}

// bayesParam wraps a model.Hyperparameter to implement the bayesopt.Param interface.
type bayesParam struct {
	model.Hyperparameter
	name string
	rand *nprand.State
}

func (b bayesParam) GetName() string {
	return b.name
}

func (b bayesParam) GetMax() float64 {
	switch {
	case b.ConstHyperparameter != nil:
		return b.ToFloat(b.ConstHyperparameter.Val)
	case b.IntHyperparameter != nil:
		return b.ToFloat(b.IntHyperparameter.Maxval)
	case b.DoubleHyperparameter != nil:
		return b.ToFloat(b.DoubleHyperparameter.Maxval)
	case b.LogHyperparameter != nil:
		return b.ToFloat(b.LogHyperparameter.Maxval)
	case b.CategoricalHyperparameter != nil:
		return b.ToFloat(len(b.CategoricalHyperparameter.Vals) - 1)
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type: %+v", b.Hyperparameter))
	}
}

func (b bayesParam) GetMin() float64 {
	switch {
	case b.ConstHyperparameter != nil:
		return b.ToFloat(b.ConstHyperparameter.Val)
	case b.IntHyperparameter != nil:
		return b.ToFloat(b.IntHyperparameter.Minval)
	case b.DoubleHyperparameter != nil:
		return b.ToFloat(b.DoubleHyperparameter.Minval)
	case b.LogHyperparameter != nil:
		return b.ToFloat(b.LogHyperparameter.Minval)
	case b.CategoricalHyperparameter != nil:
		return b.ToFloat(0)
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type: %+v", b.Hyperparameter))
	}
}

func (b bayesParam) Sample() float64 {
	return b.ToFloat(sampleOne(b.Hyperparameter, b.rand))
}

func (b bayesParam) ToFloat(x interface{}) float64 {
	switch {
	case b.ConstHyperparameter != nil:
		return 0
	case b.IntHyperparameter != nil:
		return float64(x.(int))
	case b.DoubleHyperparameter != nil:
		return x.(float64)
	case b.LogHyperparameter != nil:
		return x.(float64)
	case b.CategoricalHyperparameter != nil:
		return x.(float64)
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type: %+v", b.Hyperparameter))
	}
}

func (b bayesParam) FromFloat(x float64) interface{} {
	switch {
	case b.ConstHyperparameter != nil:
		return b.ConstHyperparameter.Val
	case b.IntHyperparameter != nil:
		return int(x)
	case b.DoubleHyperparameter != nil:
		return x
	case b.LogHyperparameter != nil:
		return x
	case b.CategoricalHyperparameter != nil:
		return b.CategoricalHyperparameter.Vals[int(x)]
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type: %+v", b.Hyperparameter))
	}
}
