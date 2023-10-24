.. _topic-guides_hp-tuning-det_custom:

#######################
 Custom Search Methods
#######################

+----------------------------------------------------------------+
| API reference                                                  |
+================================================================+
| :ref:`custom-searcher-reference`                               |
+----------------------------------------------------------------+

Determined supports defining your own hyperparameter search algorithms and provides search runner
utilities for executing them.

.. tip::

   Remember that a Determined experiment is a set of trials, each corresponding to a point in the
   hyperparameter space.

To implement a custom hyperparameter tuning algorithm, subclass
:class:`~determined.searcher.SearchMethod`, overriding its event handler methods. If you want to
achieve fault tolerance and your search method carries any state in addition to the SearcherState
passed into the event handlers, also override
:meth:`~determined.searcher.SearchMethod.save_method_state` and
:meth:`~determined.searcher.SearchMethod.load_method_state`.

To run the custom hyperparameter tuning algorithm, you can use:

-  :class:`~determined.searcher.LocalSearchRunner` to run on your machine,
-  :class:`~determined.searcher.RemoteSearchRunner` to run on a Determined cluster.

.. note::

   Using :class:`~determined.searcher.RemoteSearchRunner` will create two experiments, with one
   orchestrating the hyperparameter search of the other.

Both search runners execute the custom hyperparameter tuning algorithm and start a multi-trial
experiment on a Determined cluster.

The following sections describe the steps needed to implement and use a custom hyperparameter search
algorithm.

**********************************************
 Experiment Configuration for Custom Searcher
**********************************************

Specify the ``custom`` searcher type in the experiment configuration:

.. code:: yaml

   searcher:
     name: custom
     metric: validation_loss
     smaller_is_better: true
     unit: batches

***********************************
 Run Hyperparameter Search Locally
***********************************

A script performing hyperparameter tuning using :class:`~determined.searcher.LocalSearchRunner` may
look like the following ``run_local_searcher.py``:

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

       # Instantiate your search method, passing the necessary parameters.
       search_method = MySearchMethod(...)

       search_runner = searcher.LocalSearchRunner(search_method, searcher_dir=searcher_dir)

       experiment_id = search_runner.run(model_config, model_dir=model_context_dir)
       logging.info(f"Experiment {experiment_id} has been completed.")

To start the custom search method locally, you can use the following CLI command:

.. code:: bash

   $ python run_local_searcher.py

****************************************
 Run Hyperparameter Search on a Cluster
****************************************

A script to run your custom search method on a Determined cluster may look like the following
``run_remote_searcher.py``:

.. code:: python

   import determined as det
   from pathlib import Path
   from determined import searcher

   if __name__ == "__main__":
       model_context_dir = "experiment_files"

       model_config = "experiment_files/config.yaml"

       with det.core.init() as core_context:
           info = det.get_cluster_info()
           assert info is not None

           search_method = MySearchMethod(...)

           search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)
           search_runner.run(model_config, model_dir=model_context_dir)

To start the custom search method on a cluster, you need to submit it to the master as a
single-trial experiment. To this end, you can use the following CLI command:

.. code:: bash

   $ det e create searcher_config.yaml context_dir

The custom search method runs on a Determined cluster as a single trial experiment. Configuration
for the search method experiment is specified in the ``searcher_config.yaml`` and may look like
this:

.. code:: yaml

   name: remote-searcher
   entrypoint: python3 run_remote_searcher.py
   searcher:
     metric: validation_error
     smaller_is_better: true
     name: single
     max_length:
       batches: 1000
   max_restarts: 0
