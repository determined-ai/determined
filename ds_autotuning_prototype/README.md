# DeepSpeed Autotuning Prototype: Wrapper Scripts

The code in `/dsat` is a basic prototype of how we might implement DeepSpeed Autotuning for Core API experiments.

## Main Idea

The basic steps:

1. Perform a short, single-record profiling run to get model info using DS's `FlopsProfiler`
2. Use the profiler info and the user config to determine sets of hyperparameters to test, orchestrated by a Custom Searcher.
3. Run and profile the experiments generated in 2, reporting the relevant metrics back to the searcher.

In all cases, the `FlopsProfiler` is used to collect performance metrics, which are written to a file.

## Basic Usage

In the same directory as this `README`, run the following:

```bash
python3 -m dsat.autotune autotune_config.yaml .
```

## Pros, Cons, and TODOs

Pros:

- No need to change user script when switching between DS AT and a vanilla DS AT training run.
- Custom Searcher config generated from initial user config; user need only provide one config, per usual.
- Largely independent of `DeepSpeedEngine`

Cons:

- Currently, the user must configure DS through a `ds_config` sub-dictionary within the `hyperparameters` dict.
- Dependent on precise format of the `FlopsProfiler` output format.

TODOs:

- Support workflows which initialize DS Engine through CLI args.
- Error handling largely ignored throughout.
