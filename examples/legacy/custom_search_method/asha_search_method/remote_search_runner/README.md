# Custom SearchMethod with RemoteSearchRunner

In this example, we use RemoteSearchRunner, which executes a custom SearchMethod as a single trial experiment and
orchestrates a multi-trial experiment. Both the custom SearchMethod and the multi-trial experiment are executed
on the Determined cluster.

For an example of running the custom SearchMethod locally,
see `examples/custom_search_method/asha_custom_search_method/local_search_runner`.

## Files
* **run_experiment.py**: The code for running a custom SearchMethod with RemoteSearchRunner.

### Configuration Files
* **searcher.yaml**: Configuration for running custom SearchMethod as an experiment on the Determined cluster.


## To Run
If you have not yet installed Determined, installation instructions can be found
under `docs/install-admin.html` or at https://docs.determined.ai/latest/index.html

1. Set the `DET_MASTER` environment variable, which is the network address of the Determined master.
For instance, `export DET_MASTER=<master_host:port>`.
2. Run the following command in the `asha_search_method` directory to start RemoteSearchRunner on the Determined cluster:
`det experiment create remote_search_runner/searcher.yaml .`.

## Result
RemoteSearchRunner is submitted to the Determined master as a single trial experiment.
While running on the cluster, RemoteSearchRunner executes the custom SearchMethod and starts a multi-trial experiment
for hyperparameter search. Similarly to LocalSearchRunner, RemoteSearchRunner handles the communication between the
custom SearchMethod and the multi-trial experiment.