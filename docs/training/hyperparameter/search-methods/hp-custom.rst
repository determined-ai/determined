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

Subclass :class:`~determined.searcher.SearchMethod`. Override its event handlers. If your search
method carries any state in addition to SearcherState passed into the event handlers and if you want
to achieve fault tolerance, override :meth:`~determined.searcher.SearchMethod.save_method_state` and
:meth:`~determined.searcher.SearchMethod.load_method_state` of
:class:`~determined.searcher.SearchMethod`.

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
       model_context_dir = "experiment_files"

       model_config = "experiment_files/config.yaml"

       with det.core.init() as core_context:
           info = det.get_cluster_info()
           assert info is not None

           search_method = MySearchMethod(...)

           search_runner = searcher.RemoteSearchRunner(search_method, context=core_context)
           search_runner.run(model_config, model_dir=model_context_dir)
