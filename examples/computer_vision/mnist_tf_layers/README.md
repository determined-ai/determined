# Training a TensorFlow Graph in Determined (via Estimator API)

This example shows how wrap a graph defined in low-level TensorFlow APIs in a
custom Estimator, and then run it in Determined.

## Files
* **model_def.py**: The core code for the model.  This includes code for
defining the model in low-level TensorFlow APIs, as well as for defining the
custom Estimator and the EstimatorTrial.

* **startup-hook.sh**: Predownload the dataset in the container.  This ensures
that the dataset download does not cause conflicts between multiple workers
trying to download to the same directory if you were to reconfigure the
experiment for distributed training.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.

## Data
Estimators require tf.data.Datasets as inputs.  This examples uses the
`tensorflow_datasets` MNIST dataset as input.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f
const.yaml .`. The other configurations can be run by specifying the appropriate
configuration file in place of `const.yaml`.

## Results
Training the model with the hyperparameter settings in `const.yaml` should yield
a validation error of < 2%.
