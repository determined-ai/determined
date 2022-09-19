"""
This script runs a custom SearchMethod with CoreSearchRunner on a Determined cluster.

CoreSearchRunner is responsible for:
 -> creating a multi-trial hyperparameter search experiment in the Determined cluster,
 -> executing the custom SearchMethod as a single trial experiment in the Determined cluster,
 -> handling communication between the multi-trial experiment and the custom SearchMethod,
 -> enabling fault tolerance for SearchMethods that implement save_method_state() and
    load_method_state().

CoreSearchRunner receives SearcherEvents from the multi-trial experiment, and passes
the events to your custom SearchMethod, which, in turn, produces a list of Operations.
Next, CoreSearchRunner sends the operations to the multi-trial experiment for execution.
"""

import logging
import determined as det
from asha import ASHASearchMethod
from determined.searcher.core_search_runner import CoreSearchRunner

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)


    # Path to the .yaml file with experiment configuration
    experiment_config = 'custom_config.yaml'

    ########################################################################
    # Fault Tolerance for CoreSearchRunner
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    #
    # To support fault tolerance, CoreSearchRunner saves information required to
    # resume CoreSearchRunner and SearchMethod in the case of unexpected process termination.
    # The required information is stored in CoreSearchRunner experiment checkpoint (reported through Core API
    # checkpointing) and includes experiment id, CoreSearchRunner state, and SearchMethod state.
    # To learn more about checkpointing, see https://docs.determined.ai/latest/reference/python-api.html#checkpoint
    #
    # While CoreSearchRunner saves its own state and ensures invoking save() and
    # load() methods when necessary, a user is responsible for implementing
    # SearchMethod.save_method_state() and SearchMethod.load_method_state() to ensure correct
    # resumption of the SearchMethod.
    with det.core.init() as core_context:

        # Instantiate your implementation of SearchMethod
        search_method = ASHASearchMethod(max_length=1000, max_trials=16, num_rungs=3, divisor=4)

        # Instantiate CoreSearchRunner
        search_runner = CoreSearchRunner(search_method, context=core_context)

        ########################################################################
        # Run CoreSearchRunner
        # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
        # 1) Creates new experiment or loads state:
        #      -> if checkpoint for an experiment exists, then CoreSearchRunner loads its own state
        #         and invokes SearchMethod.load_method_state() to restore SearchMethod state;
        #      -> otherwise, new experiment is created.
        # 2) Handles communication between the multi-trial experiment and the custom SearchMethod
        # 3) Exits when the experiment is completed.
        search_runner.run(experiment_config)
