# TensorFlow (Estimator API) Boosted Trees Example

This example shows how to build a boosted trees model on the MNIST dataset using
Determined's TensorFlow Estimator API. This example is adapted from this [TensorFlow
Estimator example](https://www.tensorflow.org/tutorials/estimator/boosted_trees).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data
The current implementation uses the titanic dataset downloaded from TensorFlow APIs.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation accuracy of ~74%.
