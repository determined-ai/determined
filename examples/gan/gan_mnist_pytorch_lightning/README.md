# PyTorch Lightning MNIST GAN Example

This example demonstrates how to build a simple GAN on the MNIST dataset using
Determined's PyTorch Lightning API. This example is adapted from this [PyTorch Lightning GAN
example](https://colab.research.google.com/github/PytorchLightning/pytorch-lightning/blob/master/notebooks/03-basic-gan.ipynb#scrollTo=3vKszYf6y1Vv).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as const.yaml, but instead uses multiple GPUs (distributed training).

## To Run
Installation instructions can be found under `docs/install-admin.html` or at [Determined installation page](https://docs.determined.ai/latest/index.html).
After configuring the settings in `const.yaml`, run the following command: `det -m <master host:port> experiment create -f const.yaml . `
