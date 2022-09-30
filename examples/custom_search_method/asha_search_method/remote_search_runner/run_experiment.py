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
import determined as det
from asha import ASHASearchMethod
from utils import sample_params
from attrdict import AttrDict
from determined import searcher


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
    # To learn more about checkpointing, see https://docs.determined.ai/latest/reference/python-api.html#checkpoint
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
        args = AttrDict(info.trial.hparams)

        # Instantiate your implementation of SearchMethod
        search_method = ASHASearchMethod(
            search_space=sample_params,
            max_length=args.max_length,
            max_trials=args.max_trials,
            num_rungs=args.num_rungs,
            divisor=args.divisor,
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
