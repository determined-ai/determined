# TensorFlow (tf.keras) CIFAR-10 CNN Example

This example shows how to build a simple CNN on the CIFAR-10 dataset using
Determined's tf.keras API. This example is adapted from this [Keras CNN
 example](https://github.com/fchollet/keras/blob/master/examples/cifar10_cnn.py).

## Files
* **model_def.py**: Organizes the model and data-loaders into the Determined TFKerasTrial API.
* **cifar_model.py**: The core code for the model. This includes building and compiling the model.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values. 
* **distributed.yaml**: Same as `const.yaml`, but instead uses multiple GPUs.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm. 

## Data:
The current implementation uses CIFAR-10 data downloaded from AWS S3.

## To Run:
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
const.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `const.yaml`.

## Results:
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~74%.
