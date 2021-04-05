# PyTorch MNIST CNN Tutorial
This tutorial shows how to build a simple CNN on the MNIST dataset using
Determined's PyTorch API and showcases PyTorchTrial checkpoint callbacks.
This example is adapted from this [PyTorch MNIST
tutorial](https://github.com/pytorch/examples/tree/master/mnist).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.
* **layers.py**: Defines the convolutional layers that the model uses. 

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data
The current implementation uses MNIST data downloaded from AWS S3.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
const.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~97%. 
