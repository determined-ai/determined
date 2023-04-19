"""
This script runs a custom SearchMethod with RemoteSearchRunner on a Determined cluster.

RemoteSearchRunner is responsible for:
 -> executing the custom SearchMethod as a single trial experiment on the Determined cluster,
 -> creating a multi-trial experiment on the Determined cluster,
 -> handling communication between the multi-trial experiment and the custom SearchMethod,
 -> enabling fault tolerance for SearchMethods that implement save_method_state() and
    load_method_state().

RemoteSearchRunner receives SearcherEvents from the multi-trial experiment, and passes
the events to your custom SearchMethod, which, in turn, produces a list of Operations.
Next, RemoteSearchRunner sends the operations to the multi-trial experiment for execution.
"""
import sys

sys.path.append(".")

import logging
import random
from typing import Dict

from asha import ASHASearchMethod

import determined as det
from determined import searcher


############################################################################
# User-defined function that generates a combination of hyperparameters for each trial.
# The hyperparameters are passed to a trial in the Create operation.
# In this example, the model (defined in experiment_files/model_def.py) is expecting
# the following hyperparameters:
#   -> global_batch_size,
#   -> n_filters1
#   -> n_filters2,
#   -> learning_rate,
#   -> dropout,
#   -> dropout2.
def sample_params() -> Dict[str, object]:
    hparams = {
        "global_batch_size": 64,
        "n_filters1": random.randint(8, 64),
        "n_filters2": random.randint(8, 72),
        "learning_rate": 10 ** random.uniform(-4.0, 0.0),
        "dropout1": random.uniform(0.2, 0.8),
        "dropout2": random.uniform(0.2, 0.8),
    }
    return hparams


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)

    ########################################################################
    # Multi-trial experiment
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    #
    # The content of the following directory is uploaded to Determined cluster.
    # It should include all files necessary to run the experiment (as usual).
    model_context_dir = "experiment_files"

    # Path to the .yaml file with the multi-trial experiment configuration.
    model_config = "experiment_files/config.yaml"

    ########################################################################
    # Fault Tolerance for RemoteSearchRunner
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    #
    # To support fault tolerance, RemoteSearchRunner saves information required to
    # resume RemoteSearchRunner and SearchMethod in the case of unexpected process termination.
    # The required information is stored in RemoteSearchRunner experiment checkpoint (reported through Core API
    # checkpointing) and includes experiment id, RemoteSearchRunner state, and SearchMethod state.
    # To learn more about checkpointing, see https://docs.determined.ai/latest/reference/python-sdk.html#checkpoint
    #
    # While RemoteSearchRunner saves its own state and ensures invoking save() and
    # load() methods when necessary, a user is responsible for implementing
    # SearchMethod.save_method_state() and SearchMethod.load_method_state() to ensure correct
    # resumption of the SearchMethod.
    #
    # To ensure that SearchRunner process is resumed automatically in the case of failure,
    # make sure to set `max_restarts` in the `searcher.yaml` file to a number greater than 0.

    with det.core.init() as core_context:
        info = det.get_cluster_info()
        assert info is not None

        # Instantiate your implementation of SearchMethod
        search_method = ASHASearchMethod(
            search_space=sample_params,
            max_length=1000,
            max_trials=16,
            num_rungs=3,
            divisor=4,
        )

        # Instantiate RemoteSearchRunner
        search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)

        ########################################################################
        # Run RemoteSearchRunner
        # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
        # 1) Creates new experiment or loads state:
        #      -> if checkpoint for an experiment exists, then RemoteSearchRunner loads its own state
        #         and invokes SearchMethod.load_method_state() to restore SearchMethod state;
        #      -> otherwise, new experiment is created.
        # 2) Handles communication between the multi-trial experiment and the custom SearchMethod
        # 3) Exits when the experiment is completed.
        search_runner.run(model_config, model_dir=model_context_dir)
