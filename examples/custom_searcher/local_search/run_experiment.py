"""
This script runs your custom SearchMethod and LocalSearchRunner on
your local machine, while the experiment is executed in a Determined cluster.

LocalSearchRunner is responsible for:
 -> creating an experiment in a Determined cluster,
 -> executing your SearchMethod implementation on a local machine,
 -> handling communication between the remote experiment and your local SearchMethod.

LocalSearchRunner receives SearcherEvents from the remote experiment, and passes
the events to your custom SearchMethod, which, in turn, produces a list of Operations in
response to the events. LocalSearchRunner sends the operations to the remote experiment
for execution.
"""

import logging
from asha import ASHASearchMethod
from pathlib import Path
from determined.searcher.search_runner import LocalSearchRunner

if __name__ == "__main__":

    ########################################################################
    # Context directory for LocalSearchRunner
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    #
    # The content of context directory is uploaded to Determined cluster.
    # It should include all files necessary to run the experiment (as usual).
    context_dir = 'context_dir'

    # Path to the .yaml file with experiment configuration
    model_config = "custom_config.yaml"

    ########################################################################
    # Fault Tolerance for LocalSearchRunner
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    #
    # To support fault tolerance, LocalSearchRunner saves information required to
    # resume LocalSearchRunner and SearchMethod in the case of unexpected process termination.
    # The required information is stored in LocalSearchRunner.searcher_dir and include
    # experiment id, LocalSearchRunner state, and SearchMethod state.
    #
    # While LocalSearchRunner saves its own state and ensures invoking save() and
    # load() methods when necessary, a user is responsible for implementing
    # SearchMethod.save_state() and SearchMethod.load_state() to ensure correct
    # resumption of the SearchMethod.
    searcher_dir = Path('searcher_dir')

    # Instantiate your implementation of SearchMethod
    search_method = ASHASearchMethod(max_length=1000, max_trials=16, num_rungs=3, divisor=4)

    # Instantiate LocalSearchRunner
    search_runner = LocalSearchRunner(search_method, searcher_dir=searcher_dir)

    ########################################################################
    # Run LocalSearchRunner
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    # 1) Create new experiment or load state:
    #      -> if searcher_dir contains information required for resumption
    #         (e.g., experiment id), then LocalSearchRunner loads its own state
    #         and invokes SearchMethod.load() to restore SearchMethod state;
    #      -> otherwise, new experiment is created.
    # 2) Handle communication between the remote experiment and the SearchMethod
    # 3) Exits when the experiment is completed.
    experiment_id = search_runner.run(model_config, context_dir=context_dir)
    logging.info(f"Experiment {experiment_id} has been completed.")

