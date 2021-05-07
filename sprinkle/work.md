# Push Architecture: Order of Work

What things must happen to build the push architecture?  Note that push
architecture is a pre-requisite for Sprinkle API, and it is also closely
related to the low-level Sprinkle API, but the work outlined here does not
actually deliver either of the low-level or high-level Sprinkle APIs.

### Master-side work

1. Redo Searcher Progress Calculation

   Let the workload sequencer (and eventually the harness) report its progress
   in the current searcher operation and let the searcher aggregate individual
   trials' progress into total progress.

1. Rewrite save experiment best as a simple `GetExperimentBest()`-like call
   against the database.  No more interaction with the actor system.

1. Change Searcher to emit the new SearcherOp: each op is a single combined
   training/validation/checkpoint request, identified by absolute lengths of
   how much training should have occurred.

1. Support proposed Searcher Push API, internally to the master.

   Write the functions that the REST API endpoints would call.  Don't write the
   REST API endpoints yet though.

1. Support proposed Metrics/Checkpoint push api, internal to master.

1. Support a Preemption API, internal to master.  Preemption means something
   was descheduled or paused.  Searcher-requested stops go through the searcher
   api.

1. (assume harness-side work is done) Expose all push apis and delete golang
   workload sequencer.  Migrate to refactored harness.

### Harness-side work

1. Implement a dummy master with all push APIs

1. Write a python workload sequencer

1. Refactor harness, eliminate layers in favor of push APIs

## Raw notes:
```
# rb, shiyuan, bradley

Order of Work, or "what do we need to do accomplish this design":
  (no projects depend on partial push arch; LightiningTrial and PythonTrial did, but are not active)
  - progress calculation
        necessary since progress is calcluated by searcher based on training workloads completed
        to make sprinkle api, don't want to make user push training work to searcher
        don't want to hijack metrics endpoints because that mixes concerns

        optional api for reporting progress?
            we think this is reasonable
        every time you report training metrics?
        separate api?
        try to infer it from training metrics?
            this involves the same master-knows-dataset-length problems we currently have

  - rewrite or delete: save experiment best
      - ryan votes delete this (edit, he now votes for the API call)
      - bradley,shiyuan votes trial calls an API that asks for the best validation so far.
      - edit: a new proposal is to let pushing validation metrics respond with
        the best metric so far.
            this seems like a strange appendage to the push architecture call
            though, since all the other metrics pushing would be unidirectional
            and could even be done asynchronously if you wanted.

  - change searcher uses abs lengths

  - support searcher push api, internally to master
        write functions which the REST API endpoints would call
        don't write the REST API endpoints yet
        basically: Ask the experiment for each api call

  - support metrics/checkpoint push api, interal to master
        basically: just write metrics/checkpoints to database

  - preemption API

  - modify golang workload sequencer to use searcher/metrics/checkpoint push apis
        "fucking harder than you thought" -shiyuan
        pass the latest checkpoint into the container from the db

  - move sequencer to python
        includes min_checkpoint_period
        includes min_validation_period
        python sequencer would be inside PyTorchTrial, TFKerasTrial, EstimatorTrial
            just one python sequencer though
```
