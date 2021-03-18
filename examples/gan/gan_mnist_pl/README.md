# PyTorch Lightning MNIST GAN Example

This tutorial shows how to build a GAN on the MNIST dataset using
Determined's PyTorch API and the corresponding Lightning Adapter.
This example is adapted from [PyTorch Lightning GAN
example](https://github.com/PyTorchLightning/pytorch-lightning/blob/master/pl_examples/domain_templates/generative_adversarial_net.py).

## Files
* **model_def.py**: The code for implementing `LightningAdapter` API.
* **mnist.py**: A standalone PyTorch Lightning experiment defining a `LightningModule` implementing the model.
* **data.py**: The data loading and preparation code for the model implementing `LightningDataModule`.


### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive
* **distributed.yaml**: Same as adaptive.yaml, but instead uses multiple GPUs (distributed training)
and a bigger max training epoch.

## To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).
After configuring the settings in `const.yaml`, run the following command: `det -m <master host:port> experiment create -f const.yaml . `
