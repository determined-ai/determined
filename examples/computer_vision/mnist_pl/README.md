# PyTorch Lightning MNIST CNN Example
This tutorial shows how to build a simple CNN on the MNIST dataset using
Determined's PyTorch API and the corresponding Lightning Adapter.
This example is adapted from [PyTorch MNIST tutorial](https://github.com/pytorch/examples/tree/master/mnist).

## Files
* **model_def.py**: The code for implementing `LightningAdapter` API.
* **mnist.py**: A standalone PyTorch Lightning experiment defining a `LightningModule` implementing the model.
* **data.py**: The data loading and preparation code for the model implementing `LightningDataModule`.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive
hyperparameter tuning algorithm.

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
a validation accuracy of ~96%. 
