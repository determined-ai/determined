# HuggingFace Trainer API and Determined

The examples in this directory demonstrate how to use Determined callback with Hugging Face Trainer API to
enable Determined's distributed training, fault tolerance, checkpointing and metrics reporting.

The main callback is located in `det_callback.py` and the associated `DetCallback` object is used
in model code as in:

```
    det_callback = DetCallback(training_args, filter_metrics=["loss", "accuracy"], tokenizer=feature_extractor)
    trainer.add_callback(det_callback)
```

The subdirectories contain two examples adapted from the official Hugging Face training scripts:

- `image_classification/`: contains the [HF image classification trainer script](https://github.com/huggingface/transformers/tree/main/examples/pytorch/image-classification).
- `language_modeling/`: contains the [HF causal language modeling trainer](https://github.com/huggingface/transformers/tree/main/examples/pytorch/language-modeling).

## Script Files

In both `image_classification/image_classification.py` and `language_modeling/run_clm.py`, one can
find the training scripts which load a model from the HF Model Hub, configure the Trainer, and the
Determined callback.

### Configuration Files

Both subdirectories have each of the following configuration files:

- **const.yaml**: Train the model with constant hyperparameter values for a given number of batches (or `max_steps`).
- **const_epochs.yaml**: Train the model with constant hyperparameter values for a given number of epochs.
- **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).
- **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.
- **deepspeed.yaml**: Train the model with DeepSpeed with constant hyperparameter values. Feel free to modify this
  file to enable adaptive hyperparameter tuning algorithm.

Deepspeed configurations files are located in `ds_configs` and include:

- **ds_config_stage_1.json**: Optimizer state partitioning (ZeRO stage 1).
- **ds_config_stage_2.json**: Gradient partitioning (ZeRO stage 2).
- **ds_config_stage_2_cpu_offload.json**: Gradient partitioning and CPU offloading (ZeRO stage 2).
- **ds_config_stage_3.json**: Parameter partitioning (ZeRO stage 3).

To learn more about DeepSpeed, see [DeepSpeed docs](https://deepspeed.readthedocs.io/en/latest/) and
[HF DeepSpeed integration](https://huggingface.co/docs/transformers/main_classes/deepspeed).

## Data

The image classification example uses [the beans dataset](https://huggingface.co/datasets/beans),
while the language modeling example uses [the wikitext dataset](https://huggingface.co/datasets/wikitext)

## To Run

If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

In order to run the classification script, `cd` into `image_classification/` and run the following
to use the `const.yaml` configk

```
det experiment create const.yaml . --include ../det_callback.py
```

The language modeling script is run similarly: `cd` instead into `language_modeling/` before entering
the above command.

Other configurations can be run by specifying the appropriate configuration file in place
of `const.yaml`. For instance, to use DeepSpeed, run

```
det experiment create deepspeed.yaml . --include ../det_callback.py
```

The deepspeed configuration can be changed by altering the `hyperparameters.deepspeed_config` entry
of the `deepspeed.yaml` config, as well as the corresponding line in the `entrypoing`. The default
configuration is `ds_configs/ds_config_stage_1.json`.

One can also use Determined's DeepSpeed Autotune functionality to autotmatically optimize the
DeepSpeed settings. From either subdirectory, run the following script:

```
python3 -m determined.pytorch.deepspeed.dsat deepspeed.yaml . --include ../det_callback.py
```

## Results

Training the image classification model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~96% after 3 epochs.
