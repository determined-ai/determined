# DeepSpeed Autotuning Prototype: Core API

The code in `/dsat` is a basic prototype of how we might implement DeepSpeed Autotuning for Core API experiments.

## Main Idea

The basic steps:

1. Perform a short, model profiling run by triggering the native DS AT profiling run and collecting
   metrics appropriately.
2. Use the profiler info and the user config to determine sets of hyperparameters to test, orchestrated by a Custom Searcher.
3. Run and profile the experiments generated in 2, reporting the relevant metrics back to the searcher.

## Basic Usage

In the `./core_api/torchvison_models/` directory, for instance, run the following:

```bash
python3 -m determined.pytorch.dsat single.yaml .
```

(the config may need to be altered for your cluster.)

## Pros, Cons, and TODOs

A very incomplete list.

Pros:

- No need to write a new config.

Cons:

- Introduces a new CLI based api.
- Dependent on precise format of DS output files, brittle.
- Relies on some DS internals (effectively) to kick off the model profiling run.

TODOs:

- Support workflows which initialize DS Engine through CLI args.
- Support option for follow-on experiment.
- Benchmark against native DS AT. The 0.8.1 update which fixed the DS AT units issues might have also
  broken the `--autotuning run` flag?
