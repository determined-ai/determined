# DeepSpeed Autotuning Prototype: Core API

The code in `/dsat` is a basic prototype of how we might implement DeepSpeed Autotuning for Core API experiments.

## Main Idea

The basic steps:

1. Perform a short, model profiling run by triggering the native DS AT profiling run and collecting
   metrics appropriately.
2. Use the profiler info and the user config to determine sets of hyperparameters to test, orchestrated by a Custom Searcher.
3. Run and profile the experiments generated in 2, reporting the relevant metrics back to the searcher.

## Basic Usage

In the same directory as this `README`, run the following:

```bash
python3 -m dsat.autotune autotune_config.yaml .
```

## Pros, Cons, and TODOs

A very incomplete list.

Pros:

- Custom Searcher config generated from initial user config; user need only provide one config, per usual.
- Largely independent of `DeepSpeedEngine`

Cons:

- Currently, the user must configure DS through a `ds_config` sub-dictionary within the `hyperparameters` dict.
- Config format doesn't mirror that of standard searchers; all search specific config lives under `hyperparameters`.
- Dependent on precise format of DS output files, brittle.
- Relies on some DS internals (effectively) to kick off the model profiling run.

TODOs:

- Support workflows which initialize DS Engine through CLI args.
- Error handling largely ignored throughout.
