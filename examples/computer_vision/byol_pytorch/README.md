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
* [model_def.py](model_def.py): Core trial and callback definitions.  This is the entrypoint for trials.
* [optim.py](optim.py): Optimizer definitions and utilities.
* [reducers.py](reducers.py): Custom reducers used for evaluation metrics.
* [startup-hook.sh](startup-hook.sh): This script will automatically be run by Determined during startup of every container launched for this experiment.  This script installs some additional dependencies.
* **utils.py**: Simple utility functions and classes.

# Configuration Files
* **const-cifar10.yaml**: Train with CIFAR-10 on a single GPU with constant hyperparameter values.
* **const-stl10.yaml**: Train with STL-10 on a single GPU with constant hyperparameter values
* **distributed-imagenet.yaml**: Train with ImageNet using 64 GPU distributed training with constant hyperparameter values.

# Data
This repo uses three datasets:
- CIFAR-10 (32x32, 10 classes), automatically downloaded via torchvision.
- STL-10 (96x96, 10 classes), automatically downloaded via torchvision.
- ImageNet-1k (1000 classes), which must be downloaded and [made available to each trial container](https://docs.determined.ai/latest/training-apis/experiment-config.html?highlight=bind#bind-mounts).  Information on downloading ImageNet-1k is available at the [ImageNet website](https://image-net.org/download.php).

# To Run
If you have not yet installed Determined, installation instructions can be found under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f config/const-cifar10.yaml .`

The other configurations can be run by specifying the appropriate configuration file in place of `const-cifar10.yaml`.

## Results

| Config file | Test Accuracy (%) |
| ----------- | ------------- |
| const-cifar10.yaml | 74.91 |
| const-stl10.yaml | 88.47 |