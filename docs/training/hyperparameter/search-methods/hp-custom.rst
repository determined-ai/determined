.. _topic-guides_hp-tuning-det_custom:

#######################
 Custom Search Methods
#######################

+----------------------------------------------------------------+
| API reference                                                  |
+================================================================+
| :doc:`/reference/reference-searcher/custom-searcher-reference` |
+----------------------------------------------------------------+

Determined supports defining your own hyperparameter search algorithms and provides search runner
utilities for executing them.

.. tip::

   Remember that a Determined experiment is a set of trials, each corresponding to a point in the
   hyperparameter space.

To implement a custom hyperparameter tuning algorithm, subclass
:class:`~determined.searcher.SearchMethod`, overriding its event handler methods. Note that
overriding :meth:`~determined.searcher.SearchMethod.save_method_state` and
:meth:`~determined.searcher.SearchMethod.load_method_state` of
:class:`~determined.searcher.SearchMethod` is not mandatory. However, it is *recommended* as it
enables fault tolerance.

To run the custom hyperparameter tuning algorithm, you can use:

-  :class:`~determined.searcher.LocalSearchRunner` that executes on your machine,
-  :class:`~determined.searcher.RemoteSearchRunner` that runs on a Determined cluster.

Both Search Runners execute the custom hyperparameter tuning algorithm and start a multi-trial
experiment on a Determined cluster.

The following sections explain the steps to take in order to implement and use a custom
hyperparameter search algorithm. A detailed example can be found here:
:download:`asha_search_method.tgz </examples/asha_search_method.tgz>`

**********************************************
 Experiment Configuration for Custom Searcher
**********************************************

You have to specify "custom" searcher type in the experiment configuration:

.. code:: yaml

   searcher:
     name: custom
     metric: validation_loss
     smaller_is_better: true
     unit: batches

******************************
 Search Method Implementation
******************************

Subclass :class:`~determined.searcher.SearchMethod`. Below is a starting template:

.. code:: python

   import json
   import uuid
   from pathlib import Path
   from typing import List

   class MySearchMethod(searcher.SearchMethod):
       def __init__(self, ...) -> None:
           super().__init__()
           # initialize any state you would need

       def initial_operations(self) -> List[searcher.Operation]:
           # Create and return the initial list of operations
           # immediately after an experiment has been created
           # Currently, we support the following operations:
           # - Create - starts a new trial with a unique trial id and a set of hyperparameters,
           # - ValidateAfter - sets the number of steps (i.e., batches or epochs) after which
           #                   a validation is run, for a trial with a given id,
           # - Close - closes a trial with a given id,
           # - Shutdown - closes the experiment.
           return []

       def on_trial_created(self, request_id: uuid.UUID) -> List[searcher.Operation]:
           # note: the request_id argument in this and other methods
           # uniquely identifies a trial
           # update state as needed
           # return operations to be performed when a trial is created
           return []

       def on_validation_completed(
           self,
           request_id: uuid.UUID,
           metric: float,
           train_length: int,
       ) -> List[searcher.Operation:
           # return operations to be performed based on the state,
           # the value of the metric returned by the validation
           # for a given trial, and the length of the training
           # (in units specified in the searcher configuration)
           return []

       def on_trial_closed(self, request_id: uuid.UUID) -> List[searcher.Operation]:
           # update internal state, reflecting the completion of the trial
           # identified by request_id
           # return operations
           return []

       def progress(self) -> float:
           # report experiment progress as a value between 0.0 and 1.0
           # the Web UI will display a corresponding progress bar
           return 0.0

       def on_trial_exited_early(self) -> List[searcher.Operation]:
           # update internal state, reflecting early trial exit
           # return operations (e.g., create a trial with a different
           # combination of hyperparameters)
           return []

       def save_method_state(self, path: Path) -> None:
           # save any useful state to a file you create in directory path
           checkpoint_path = path.joinpath("method_state.json")
           with checkpoint_path.open("w") as f:
               # populate a dictionary or another serializable data structure
               # with the internal state data
               # you can use any serialization format (not just json)
               state = {}
               json.dump(state, f)

       def load_method_state(self, path: Path) -> None:
           checkpoint_path = path.joinpath("method_state.json")
           with checkpoint_path.open("r") as f:
               state = json.load(f)
               # initialize internal state from the deserialized data structure

***********************************
 Run Hyperparameter Search Locally
***********************************

A script performing hyperparameter tuning using :class:`~determined.searcher.LocalSearchRunner` may
look like the following:

.. code:: python

   import logging
   from pathlib import Path
   from determined import searcher


   if __name__ == "__main__":
       # The content of the following directory is uploaded to Determined cluster.
       # It should include all files necessary to run the experiment (as usual).
       model_context_dir = "experiment_files"

       # Path to the .yaml file with the multi-trial experiment configuration.
       model_config = "experiment_files/config.yaml"

       # While LocalSearchRunner saves its own state and ensures invoking save() and
       # load() methods when necessary, a user is responsible for implementing
       # SearchMethod.save_method_state() and SearchMethod.load_method_state() to ensure
       # correct resumption of the SearchMethod.
       searcher_dir = Path("local_search_runner/searcher_dir")

       # instantiate your search method, passing the necessary parameters
       search_method = MySearchMethod(...)

       search_runner = searcher.LocalSearchRunner(search_method, searcher_dir=searcher_dir)

       experiment_id = search_runner.run(model_config, model_dir=model_context_dir)
       logging.info(f"Experiment {experiment_id} has been completed.")

****************************************
 Run Hyperparameter Search on a Cluster
****************************************

A script to run your custom search method on a Determined cluster may look like this:

.. code:: python

   import determined as det
   from pathlib import Path
   from determined import searcher

   if __name__ == "__main__":
       # The content of the following directory is uploaded to Determined cluster.
       # It should include all files necessary to run the experiment (as usual).
       model_context_dir = "experiment_files"

       # Path to the .yaml file with the multi-trial experiment configuration.
       model_config = "experiment_files/config.yaml"

       with det.core.init() as core_context:

           info = det.get_cluster_info()
           assert info is not None
           args = AttrDict(info.trial.hparams)

           # Instantiate your implementation of SearchMethod
           search_method = MySearchMethod(...)

           # Instantiate and execute RemoteSearchRunner
           search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)
           search_runner.run(model_config, model_dir=model_context_dir)
