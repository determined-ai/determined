# DeepSpeed Autotuning

This example demonstrates how to use the DeepSpeed Autotune (`dsat`) feature with two parallel examples, one
written as a [`DeepSpeedTrial`](https://docs.determined.ai/latest/training/apis-howto/deepspeed/overview.html) class and the other written using [Core API](https://docs.determined.ai/latest/training/apis-howto/api-core-ug.html#core-api).
The relevant code can be found under `deepspeed_trial/` and `core_api/`, respectively. Each example
trains a [`torchvision`](https://pytorch.org/vision/stable/models.html) model on randomly generated
ImageNet-like data (for speed and simplicity).

## Files

The two subdirectories closely mirror each other. Both contain identical `ds_config.json` files which
use a simple zero-1 DeepSpeed (DS) configuration. They also contain nearly identical`deepspeed.yaml` Determined
configuration files: `core_api/deepspeed.yaml` only differs from `deepspeed_trial/deepspeed.yaml`
in its entrypoint and the inclusion of parameters which control the metric-reporting and checkpointing
frequencies.

Model code can be found in the following files:

- `deepspeed_trial/model_def.py` contains the `DeepSpeedTrial` subclass. The only `dsat`-specific code
  in this files comes is the `dsat.get_ds_config_from_hparams` helper function which makes its easy
  to inject the `dsat`-generated parameters into the training loop.
- `core_api/script.py` contains a bare-bones training loop written with Core API. The script handles
  preemption, metric-reporting, and checkpointing. In addition to the `dsat.get_ds_config_from_hparams`
  helper function, the forward and backward steps are wrapped in the `dsat.dsat_reporting_context`
  context manager. In general, these are the only changes which are needed to make Core API code
  `dsat` compatible.

The `deepspeed.yaml` files define standard single-Trial experiments which can be run in the usual way
by calling

```bash
python3 -m determined.pytorch.dsat binary deepspeed.yaml .
```

after `cd`-ing into the relevant directory. The code path which utilizes `dsat` is described in the
following section.

## Basic Usage

There are three available search methods for DeepSpeed Autotune:

- `asha`: uses the [ASHA](https://docs.determined.ai/latest/training/hyperparameter/search-methods/hp-adaptive-asha.html#id1)
  algorithm to adaptively search over randomly selected DeepSpeed configurations, using the number of
- `binary`: tunes the optimal batch size for a handful of randomly generated DeepSpeed configurations
  via binary search.
- `random`: performs a search over randomly generated DeepSpeed configurations which implements
  aggressive early-stopping criteria based on domain-knowledge of DeepSpeed and the search history.

  After `cd`-ing into either of the two subdirectories above, a `asha dsat` experiment can be launched
  by entering the following command, for instance:

```bash
python3 -m determined.pytorch.dsat asha deepspeed.yaml .
```

with similar commands for `binary` and `random`.

## What to Expect

The DeepSpeed Autotune Feature is built on top of [Custom Searcher](https://docs.determined.ai/latest/training/hyperparameter/search-methods/hp-custom.html#custom-search-methods)
which starts up two separate Experiments:

- A `single` search runner Experiment whose role is to coordinate and schedule the actual Trials which are run.
- A `custom` Experiment which contains the Trials whose results are reported back to the search runner above.

The logs of the `single` search runner experiment will contain information regarding the size of the model,
the GPU memory available, the activation memory required per example, and an approximate computation
of the maximum batch size per zero stage. When a best-performing DS configuration is found, the
corresponding `json` configuration file will be written to the search runner's checkpoint directory,
along with a file detailing the configuration's corresponding metrics.

The `custom` experiment instead holds the visualization and summary tables which outline the results for
every Trial. Initially, a one-step profiling Trial is created to gather the above information regarding
the model and available hardware. Subsequently, multiple short Trials are submitted which each report
back metrics such as `FLOPS_per_gpu`, `throughput` (samples/second), and latency timing information.

## Arguments

By default, `dsat` launches 50 Trials and runs up to 16 concurrently. These values can be changed via
the `--max-trials` and `--max-concurrent-trials` flags. There is also an option to limit the number
of Trials by specifying `--max-slots`. Other notable flags include:

- `--metric`: specifies the metric to be optimized. Defaults to `FLOPS_per_gpu`. Other available options
  are `throughput`, `forward`, `backward`, and `latency`.
- `--run-full-experiment`: When this flag is specified, after every `dsat` Trial has completed, a
  single-Trial experiment will be launched using the specifications in the `deepspeed.yaml` overwritten
  with the best-found DS configuration parameters.
- `--zero-stages`: by default, `dsat` will search over each of stages `1, 2, and 3`. This flag allows the
  user to limit the search to a subset of the stages by providing a space-separated list, as in `--zero-stages 2 3`

The full options for each `dsat` search method can be found as in `python3 -m determined.pytorch.dsat binary --help` and similar for the other search methods.

(#TODO: link back to `dsat` docs)
