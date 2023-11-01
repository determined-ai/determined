# PyTorch MNIST CNN Tutorial
This tutorial shows how to build a simple CNN on the MNIST dataset using
Determined's PyTorch Trainer API. This example is adapted from this [PyTorch MNIST
tutorial](https://github.com/pytorch/examples/tree/master/mnist).

## Files
* **model.py**: The model definition and custom layers.
* **data.py**: Code for fetching and pre-processing data for the model.
* **train.py**: Implementation of the core training loop workflow, the entrypoint for training the model.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).
* **dist_random.yaml**: Distributed training with a random grid search algorithm.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data
This examples uses the MNIST dataset from the `torchvision` datasets subpackage. See 
[torchvision docs](https://pytorch.org/vision/main/generated/torchvision.datasets.MNIST.html#torchvision.datasets.MNIST) 
for details.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

The training loop is invoked through the 
[`Trainer.fit()`](https://docs.determined.ai/latest/reference/training/api-pytorch-reference.html#determined.pytorch.Trainer.fit) 
method, which accepts various arguments for configuring training behavior. The example shown in `train.py` 
illustrates a single implementation that can run in two modes (local training and on-cluster) without
any code changes.

### Local Training
The training code in `train.py` can be invoked locally as a regular Python script. Configure the appropriate training 
lengths, checkpoint/validation periods, or other desired local training functionality in the `Trainer.fit()` call, 
then run `python3 train.py` from your local environment.

### On-cluster
To run training on-cluster, configure the desired training arguments (checkpoint or validation periods, 
checkpoint to start from, etc.) in the `Trainer.fit()` call. An experiment configuration file is also required 
for on-cluster experiments (several examples are included in the directory).

Then the code can be submitted to Determined for on-cluster training by running this command from the current directory:
`det experiment create const.yaml .`. The other configurations can be run by specifying the desired 
configuration file in place of `const.yaml`.

#### Distributed Training
To train on-cluster across multiple nodes, `slots_per_trial` and `entrypoint`must be configured in the experiment configuration. 
`entrypoint` should wrap `train.py` with a Determined launch layer module, which will launch the training script across 
the slots specified. The launch layer module can be used in single-slot trials as well, to avoid configuration changes 
between iterations.

```yaml
...
resources:
  slots_per_trial: 2
entrypoint: python3 -m determined.launch.torch_distributed python3 train.py
```

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~97%. 
