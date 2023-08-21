.. _topic-guides_hp-constraints-det:

###################################
 Hyperparameter Search Constraints
###################################

Determined's Hyperparameter (HP) Search Constraints API enables finer-grained control over the
hyperparameter search space through the specification of additional constraints. This functionality
is particularly useful for incorporating prior knowledge/domain expertise into a hyperparameter
search and constraining the search to models that fit a particular deployment environment.

Using Determined's HP Search Constraints requires no changes to the configuration files. Rather,
users can simply raise a ``determined.InvalidHP`` exception in their model code when the trial is
first created in its constructor or at any subsequent point during training. This user-raised
exception is then handled by Determined's system internally – resulting in the graceful stop of the
current trial being trained, logging the InvalidHP exception in the trial logs, and propagating that
information to the search method.

.. warning::

   It is important to note that each search method has different behavior when a
   ``determined.InvalidHP`` is raised by the user in accordance with the internal dynamics of each
   searcher, as detailed below.

***********************************************
 HP Search Constraints in PyTorch vs. TF Keras
***********************************************

Since the PyTorch and TF Keras APIs have different behavior, the timing/placement of user-raised
InvalidHP exceptions are somewhat different.

In the case of PyTorch, this exception can be raised in the trial's ``__init__``, ``train_batch``,
or ``evaluate_batch`` methods. In the case of TF Keras, this exception can be raised in the
``__init__`` method or in an ``on_checkpoint_end`` callback.

See the `hp_constraints_mnist_pytorch
<https://github.com/determined-ai/determined/tree/master/examples/features/hp_constraints_mnist_pytorch>`_
example for a demonstration of HP Search Constraints with PyTorch.

******************************************************
 Searcher-Specific Behavior for HP Search Constraints
******************************************************

.. list-table::
   :header-rows: 1

   -  -  Search Algorithm
      -  HP Search Constraints Behavior

   -  -  Single
      -  Not applicable to HP Search Constraints as only a single hyperparameter configuration will
         be trained.

   -  -  Grid
      -  Does nothing since grid does not take actions based on search status or progress.

   -  -  Random
      -  Gracefully terminates current trial, creates a new trial with a randomly sampled set of
         hyperparameters and adds it to the trial queue.

   -  -  Adaptive (ASHA)
      -  Gracefully terminates and removes metrics associated with the current trial and creates a
         new trial with a randomly sampled set of hyperparameters.

************
 Next Steps
************

-  :ref:`Experiment Configuration <experiment-configuration_searcher>`
