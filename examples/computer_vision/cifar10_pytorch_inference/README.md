# PyTorch CIFAR-10 CNN Example

This example shows how to run inference with a simple CNN trained on the CIFAR-10 dataset.
This example is adapted from this [Keras CNN
example](https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py).

## Files
* **model_def.py**: The core code for the model. This includes downloading the model and defining
validation data and the inference step.

### Configuration Files
* **const.yaml**: Evaluate the model with constant hyperparameter values.

## Data
The CIFAR-10 dataset is downloaded from https://www.cs.toronto.edu/~kriz/cifar.html.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
Running inference the model with the hyperparameter settings in `const.yaml` should yield
a validation error of ~6%.
