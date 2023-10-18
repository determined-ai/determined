# Pytorch Bootstrap Your Own Latent (BYOL) Example

This example shows how to perform self-supervised image classifier training with BYOL using
Determined's PyTorch API.  This example is based on the [byol-pytorch](https://github.com/lucidrains/byol-pytorch/tree/master/byol_pytorch) package.

Original BYOL paper: https://arxiv.org/abs/2006.0

Code and configuration details also sourced from the following BYOL implementations:
  - (JAX, paper authors) https://github.com/deepmind/deepmind-research/tree/master/byol
  - (Pytorch) https://github.com/untitled-ai/self_supervised

# Files
* [backbone.py](backbone.py): Backbone registry.
* [data.py](data.py): Dataset downloading and metadata registry.
* [evaluate_result.py](evaluate_result.py): Kicks off an evaluation run, for longer training of classifier heads.
* [generate_blob_list.py](generate_blob_list.py): Script to generate a blob list from a GCS bucket + prefix.  Used to support GCS streaming for ImageNet dataset.
* [model_def.py](model_def.py): Core trial and callback definitions.  This is the entrypoint for trials.
* [optim.py](optim.py): Optimizer definitions and utilities.
* [reducers.py](reducers.py): Custom reducers used for evaluation metrics.
* [startup-hook.sh](startup-hook.sh): This script will automatically be run by Determined during startup of every container launched for this experiment.  This script installs some additional dependencies.
* [utils.py](utils.py): Simple utility functions and classes.

# Configuration Files
* [const-cifar10.yaml](const-cifar10.yaml): Train with CIFAR-10 on a single GPU with constant hyperparameter values.
* [distributed-stl10.yaml](distributed-stl10.yaml): Train with STL-10 using 8 GPU distributed training with constant hyperparameter values.
* [distributed-imagenet.yaml](distributed-imagenet.yaml): Train with ImageNet using 64 GPU distributed training with constant hyperparameter values.

# Data
This repo uses three datasets:
- CIFAR-10 (32x32, 10 classes), automatically downloaded via torchvision.
- STL-10 (96x96, 10 classes), automatically downloaded via torchvision.
- ImageNet-1k (1000 classes), which must stored in a GCS bucket along with a blob index.  Information on downloading ImageNet-1k is available at the [ImageNet website](https://image-net.org/download.php).  See `distributed-imagenet.yaml` for an example bucket configuration, and `generate_blob_list.py` for a script to generate the blob list.

# To Run
If you have not yet installed Determined, installation instructions can be found under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command to kick off self-supervised training: `det -m <master host:port> experiment create -f config/const-cifar10.yaml .`

The other configurations can be run by specifying the appropriate configuration file in place of `const-cifar10.yaml`.


To run classifier training and validation on a completed self-supervised training:

1. Find the experiment ID of your self-supervised training.
2. Run `python evaluate_result.py --experiment-id=<id> --classifier-train-epochs=<number>`

This is necessary for ImageNet, where `hyperparameters.validate_with_classifier` is set to `false` during self-supervised training due to the time it takes to train the classifier.  Other configs have `hyperparameters.validate_with_classifier` set to true to collect `test_accuracy` during the self-supervised training.


## Results

For `const-cifar10.yaml` and `distributed-stl10.yaml`, results were taken from best `test_accuracy` achieved over the self-supervised training duration.  For `distributed-imagenet.yaml`, result was taken from running `evaluate_result.py` for 80 classifier training epochs.

| Config file | Test Accuracy (%) |
| ----------- | ------------- |
| const-cifar10.yaml | 74.91 |
| distributed-stl10.yaml | 91.10 |
| distributed-imagenet.yaml | 71.37 |