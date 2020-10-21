# TensorFlow (tf.keras) Iris Species Categorization Example

This example shows how to run a CNN on the Iris species dataset using
Determined's tf.keras API. This example is adapted from this [Iris species 
categorization medium post](https://medium.com/@nickbortolotti/iris-species-categorization-using-tf-keras-tf-data-and-differences-between-eager-mode-on-and-off-9b4693e0b22).

## Files
* **model_def.py**: The core code for the model. This includes building and compiling the model.
* **startup-hook.sh**: Additional dependencies that Determined will automatically install into each container for this experiment.

### Configuration Files
* **const.yaml**: Train the model with constant hyperparameter values.
* **adaptive.yaml**: Perform a hyperparameter search using Determined's state-of-the-art adaptive hyperparameter tuning algorithm.

## Data:
The current implementation uses [UCI's Iris Data Set](https://archive.ics.uci.edu/ml/datasets/iris).

## To Run:
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

Run the following command: `det -m <master host:port> experiment create -f 
const.yaml .`. The other configurations can be run by specifying the appropriate 
configuration file in place of `const.yaml`.

## Results:
Training the model with the hyperparameter settings in `const.yaml` should yield 
a validation accuracy of ~95%.
