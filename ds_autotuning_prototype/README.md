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
- The `FlopsProfiler` does not estimate the activation memory per GPU, which is used in native DS AT, and the lack of
  this info may put our searcher at a disadvantage. However, the subsequent computations native DS AT uses with this info
  are sometimes sketchy, so I'm not sure this is a big loss.
- Dependent on precise format of the `FlopsProfiler` output format.
- Currently, the Custom Searcher launches wrapper Trials (which are collected into a single experiment) which in turn
  launch Trials (which are not collected into a single experiment). This is messy and pollutes the Web UI.
  This general phenomenon seems hard to avoid in a clean way.

TODOs:

- Support workflows which initialize DS Engine through CLI args.
- Clean up the many unorganized trials which are generated.
- Error handling largely ignored throughout.
- The custom searcher currently runs dummy trials w/ the original config. Logic for non-trivial
  searchers not yet implemented.
- No hardware information passed to Custom Searcher, yet.
