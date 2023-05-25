# DeepSpeed Autotuning

This example demonstrates how to use the DeepSpeed Autotune (`dsat`) feature with two parallel examples, one
written as a [`DeepSpeedTrial`](https://docs.determined.ai/latest/training/apis-howto/deepspeed/overview.html)
class and the other written using [Core API](https://docs.determined.ai/latest/training/apis-howto/api-core-ug.html#core-api).
The relevant code can be found under `deepspeed_trial/` and `core_api/`, respectively. Each example
trains a [`torchvision`](https://pytorch.org/vision/stable/models.html) model on randomly generated
ImageNet-like data (for speed and simplicity).

## Files

The two subdirectories closely mirror each other.

Both contain identical `ds_config.json` files which
use a simple zero-1 DeepSpeed (DS) configuration. They also contain nearly identical`deepspeed.yaml` Determined
configuration files: `core_api/deepspeed.yaml` only differs from `deepspeed_trial/deepspeed.yaml`
in its entrypoint and the inclusion of parameters which control the metric-reporting and checkpointing
frequencies.

Model code can be found in the following files:

- `deepspeed_trial/model_def.py` contains the `DeepSpeedTrial` subclass. The only `dsat`-specific code
  in this file comes is the `dsat.get_ds_config_from_hparams` helper function.
- `core_api/script.py` contains a bare-bones training loop written with Core API. The script handles
  preemption, metric-reporting, and checkpointing. In addition to the `dsat.get_ds_config_from_hparams`
  helper function, the forward and backward steps are wrapped in the `dsat.dsat_reporting_context`
  context manager.

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
  algorithm to adaptively search over randomly selected DeepSpeed configurations
- `binary`: tunes the optimal batch size for a handful of randomly generated DeepSpeed configurations
  via binary search.
- `random`: performs a search over randomly generated DeepSpeed configurations which implements
  aggressive early-stopping criteria based on domain-knowledge of DeepSpeed and the search history.

  After `cd`-ing into either of the two subdirectories above, a `asha dsat` experiment can be launched
  by entering the following command, for instance:

```bash
python3 -m determined.pytorch.dsat asha deepspeed.yaml .
```

Similar commands are available for `binary` and `random`. The full options for each `dsat` search
method can be found as in `python3 -m determined.pytorch.dsat asha --help` and similar for the other
search methods.

See [the documentation](https://docs.determined.ai/latest/model-dev-guide/apis-howto/deepspeed/autotuning.html) for more on the available DeepSpeed Autotuning options.
