# Custom SearchMethod with LocalSearchRunner

This example shows how to implement a custom hyperparameter SearchMethod that enable fault tolerance.
To run the custom SearchMethod, in this example we use LocalSearchRunner that executes the custom SearchMethod
on your local machine. 
For an example of running the custom SearchMethod on a cluster, see `examples/custom_searcher/core_search_runner`.

## Files
* **asha.py**: The code for ASHA implemented as a custom SearchMethod.
* **run_experiment.py**: The code for running a custom SearchMethod locally with LocalSearchRunner.
* **context_dir/model_def.py**: The core code for the model. This includes building and compiling the model.
* **context_dir/data.py**: The data loading and preparation code for the model.
* **context_dir/layers.py**: Defines the convolutional layers that the model uses. 

### Configuration Files
* **custom_config.yaml**: Configuration for running `model_def.py` with a custom SearchMethod. 
Note `searcher.name: custom`.

## Data
The current implementation uses MNIST data downloaded from AWS S3.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

1. Set the `DET_MASTER` environment variable, which is the network address of the Determined master.
For instance, `export DET_MASTER=<master_host:port>`.
2. Run the following command to start the local search runner `python run_experiment.py`.

## Result
LocalSearchRunner executes the custom SearchMethod on your local machine, 
while the multi-trial experiment for hyperparameter search is started on a Determined cluster.
LocalSearchRunner handles the communication between the custom SearchMethod and the multi-trial experiment.