# TensorFlow (tf.keras) Fashion MNIST Tutorial

This tutorial shows how to build a simple CNN on the MNIST dataset using
Determined's tf.keras API. This example is adapted from this [Keras image
classification tutorial](https://www.tensorflow.org/tutorials/keras/classification).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **data.py**: The data loading and preparation code for the model.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **distributed.yaml**: Same as `const.yaml`, but trains the model with multiple GPUs (distributed training).
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data
The current implementation downloads the Fashion MNIST data from 
[here](https://github.com/zalandoresearch/fashion-mnist/blob/master/LICENSE).

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
const.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~85%. 
