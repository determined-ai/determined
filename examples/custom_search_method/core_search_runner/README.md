# Custom SearchMethod with CoreSearchRunner

This example shows how to implement a custom hyperparameter SearchMethod that enables fault tolerance.
To run the custom SearchMethod, in this example we use CoreSearchRunner that executes the custom SearchMethod
in the Determined cluster. 
For an example of running the custom SearchMethod locally, see `examples/custom_searcher/local_search_runner`.

## Files
* **context_dir/asha.py**: The code for ASHA implemented as a custom SearchMethod.
* **context_dir/run_experiment.py**: The code for running a custom SearchMethod locally with CoreSearchRunner.
* **context_dir/model_def.py**: The core code for the model. This includes building and compiling the model.
* **context_dir/data.py**: The data loading and preparation code for the model.
* **context_dir/layers.py**: Defines the convolutional layers that the model uses. 

### Configuration Files
* **custom_config.yaml**: Configuration for running `model_def.py` with a custom SearchMethod. 
Note `searcher.name: custom`.
* **searcher.yaml**: Configuration for running custom SearchMethod as an experiment in the Determined cluster. 

## Data
The current implementation uses MNIST data downloaded from AWS S3.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

1. Set the `DET_MASTER` environment variable, which is the network address of the Determined master.
For instance, `export DET_MASTER=<master_host:port>`.
2. 'cd context_dir'
3. Run the following command to start CoreSearchRunner in the Determined cluster `det experiment create searcher.yaml .`.

## Result
CoreSearchRunner is submitted to the Determined master as a single trial experiment.
While running on the cluster, CoreSearchRunner executes the custom SearchMethod and starts a multi-trial experiment
for hyperparameter search. Similarly to LocalSearchRunner, CoreSearchRunner handles the communication between the 
custom SearchMethod and the multi-trial hyperparameter search experiment.