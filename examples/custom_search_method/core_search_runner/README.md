# Custom SearchMethod with CoreSearchRunner

This example shows how to implement a custom SearchMethod that enables fault tolerance, and how to run an experiment 
with the custom SearchMethod on a Determined cluster. In this example, we use CoreSearchRunner, which executes the 
custom SearchMethod as a single trial experiment and orchestrates  a multi-trial experiment by passing operations 
from the custom SearchMethod to the experiment. Both the custom SearchMethod and the multi-trial experiment are 
executed on the Determined cluster.

For an example of running the custom SearchMethod locally, see `examples/custom_searcher/local_search_runner`.

## Files
Custom SearchMethod experiment:
* **asha.py**: The code for ASHA implemented as a custom SearchMethod.
* **run_experiment.py**: The code for running a custom SearchMethod with CoreSearchRunner.

Multi-trial experiment:
* **experiment_files/model_def.py**: The core code for the model. This includes building and compiling the model.
* **experiment_files/data.py**: The data loading and preparation code for the model.
* **experiment_files/layers.py**: Defines the convolutional layers that the model uses. 

### Configuration Files
Custom SearchMethod experiment:
* **searcher.yaml**: Configuration for running custom SearchMethod as an experiment on the Determined cluster. 

Multi-trial experiment:
* **experiment_files/custom_config.yaml**: Configuration for running `model_def.py` with a custom SearchMethod. 
Note `searcher.name: custom`.


## Data
The current implementation uses MNIST data downloaded from AWS S3.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

1. Set the `DET_MASTER` environment variable, which is the network address of the Determined master.
For instance, `export DET_MASTER=<master_host:port>`.
2. Run the following command to start CoreSearchRunner on the Determined cluster: `det experiment create searcher.yaml .`.

## Result
CoreSearchRunner is submitted to the Determined master as a single trial experiment.
While running on the cluster, CoreSearchRunner executes the custom SearchMethod and starts a multi-trial experiment
for hyperparameter search. Similarly to LocalSearchRunner, CoreSearchRunner handles the communication between the 
custom SearchMethod and the multi-trial experiment.