# Custom SearchMethod

This tutorial shows how to implement a custom SearchMethod that enables fault tolerance, and how to use a SearchRunner
to perform a hyperparameter search with the custom SearchMethod. In this example, we implemented
ASHA as a Custom Search method. SearchRunner executes ASHA implementation and orchestrates a multi-trial hyperparameter
search experiment by passing operations from the custom SearchMethod to the multi-trial experiment.

We provide two implementations of SearchRunner:
* LocalSearchRunner: executes the custom SearchMethod locally (see `local_search_runner`),
* RemoteSearchRunner: executes the custom SearchMethod on a Determined cluster (see `remote_search_runner`).

Note that, while SearchRunner and SearchMethod can be executed either on a local machine (LocalSearchRunner)
or on a cluster (RemoteSearchRunner), the multi-trial experiment is always executed on the cluster.

## Files
Custom SearchMethod:
* **asha.py**: The code for ASHA implemented as a custom SearchMethod.
* **utils.py**: The code to generate hyperparamters for ASHA.

Multi-trial experiment:
* **experiment_files/model_def.py**: The core code for the model. This includes building and compiling the model.
* **experiment_files/data.py**: The data loading and preparation code for the model.
* **experiment_files/layers.py**: Defines the convolutional layers that the model uses.


### Configuration Files
Multi-trial experiment:
* **experiment_files/config.yaml**: Configuration for running `model_def.py` with a custom SearchMethod.
Note `searcher.name: custom`. Instead of defining hyperparameters in the yaml file, each trial in the experiment 
receives hyperparameters from the custom SearchMethod.

## Data
The current implementation uses MNIST data downloaded from AWS S3.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

1. Set the `DET_MASTER` environment variable, which is the network address of the Determined master.
For instance, `export DET_MASTER=<master_host:port>`.
2. To run the experiment see:
    * `local_search_runner/README.md` to execute the custom SearchMethod locally,
    * `remote_search_runner/README.md` to execute the custom SearchMethod on a Determined cluster.


