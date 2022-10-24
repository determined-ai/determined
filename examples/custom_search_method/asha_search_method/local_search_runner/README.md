# Custom SearchMethod with LocalSearchRunner

In this example, we use LocalSearchRunner, which executes a custom SearchMethod on your local machine and
orchestrates a multi-trial experiment on a Determined cluster.

For an example of running the custom SearchMethod on a cluster,
see `examples/custom_search_method/asha_custom_search_method/remote_search_runner`.

## Files
* **run_experiment.py**: The code for running the custom SearchMethod locally with LocalSearchRunner.

## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

1. Set the `DET_MASTER` environment variable, which is the network address of the Determined master.
For instance, `export DET_MASTER=<master_host:port>`.
2. Run the following command in the `asha_search_method` directory to start LocalSearchRunner: `python local_search_runner/run_experiment.py`.

## Result
LocalSearchRunner executes the custom SearchMethod on your local machine,
while the multi-trial experiment for hyperparameter search is started on a Determined cluster.
LocalSearchRunner handles the communication between the custom SearchMethod and the multi-trial experiment.