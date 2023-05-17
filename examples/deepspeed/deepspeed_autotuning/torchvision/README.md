# DeepSpeed Autotuning

The code in `examples/deepspeed/deepspeed_autotuning` is a prototype of how we implement DeepSpeed Autotuning for Core API experiments and Deepspeed Trials in Determined.

## Basic Usage

In any of the following directories, such as `./core_api/torchvison_models/`, run the following:

```bash
python3 -m determined.pytorch.dsat deepspeed.yaml .
```

(the config may need to be altered for your cluster.)

Note: The way to run a given example may be different, be sure to check the particular example's README.

## Technical Details

The process:

1. Runs a special "single" searcher experiment whose job is to organize the DeepSpeed Hyperparameter search. 
   - This run with add a suffix to your experiment and be titled "`(DSAT) <MY_EXPERIMENT_NAME>`". We will refer to this as the "Orchestrator"
2. The orchestrator will run one trial whose job is to get a ballpark estimate of the GPU memory metrics. This is called the "model profiling run"
   - Note: It's expected that many of these trials will fail with "Out of Memory" errors. This is part of the process of testing the GPU batching memory limits.
2. The orchestrator will run a client experiment which communicates with it to run various trials with a variety of `micro_batch_size` and `gradient_accumulation_steps` (among others) to determine what runs most efficiently.
   - Note: It's expected that many of these trials will fail with "Out of Memory" errors. This is part of the process of testing the GPU batching memory limits.

## Pros, Cons, and TODOs

A very incomplete list.

Pros:

- No need to write a new config. If your example runs with deepspeed, you can run it with `determined.pytorch.dsat` instead to automatically tune it's GPU batching hyperparameters.

Cons:

- Introduces a new CLI based api.
- Dependent on precise format of DS output files, brittle.
- Relies on some DS internals to kick off the model profiling run.

TODOs:

- Support option for follow-on experiment.
