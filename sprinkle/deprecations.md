# Proposed Deprecations with Sprinkle API

As we move to no longer controlling the training loop, we should let go of
training loops that we currently own but do not deliver the user-value bang for
their engineering buck.

* `TFKerasTrial`: deprecate in favor of sprinkle-api keras support.
* `EstimatorTrial`: in favor of sprinkle-api estimator support.

It also makes sense to deprecate our training-loop-related experiment
configurations, letting them get set via their respective training loop
invocations (as args to `PyTorchContext.train()` for example).

* `hyperparameters.global_batch_size`: it's weird to have a required
  hyperparameter, but we'll finally be able to get rid of it (it will still
  be required for some training jobs for backwards compatibility).
* `min_checkpoint_period`: most training loops support this; we should let
  users configure their training loop in code, which is what they are already
  doing anyway.
* `min_validation_period`: same as for `min_checkpoint_period`
* `optimizations.aggregation_frequency`: we should expose this on the
  trainer apis where it is supported, as python rather than as a config.
* `optimizations.auto_tune_tensor_fusion`: same
* `optimizations.average_aggregated_gradients`: same
* `optimizations.average_training_metrics`: same
* `optimizations.gradient_compression`: same
* `optimizations.tensor_fusion_cycle_time`: same
* `optimizations.tensor_fusion_threshhold`: same
* `optimizations.mixed_precision`: this field is already ignored; it's only
  configurable in the PyTorchTrial via the Python API.
