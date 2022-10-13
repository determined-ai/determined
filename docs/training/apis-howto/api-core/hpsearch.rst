.. _core-hpsearch:

#######################
 Hyperparameter Search
#######################

With the Core API you can run advanced hyperparameter searches with arbitrary training code. The
hyperparameter search logic is in the master, which coordinates many different Trials. Each trial
runs a train-validate-report loop:

.. table::

   +----------+--------------------------------------------------------------------------+
   | Train    | Train until a point chosen by the hyperparameter search algorithm and    |
   |          | obtained via the Core API.  The length of training is absolute, so you   |
   |          | have to keep track of how much you have already trained to know how much |
   |          | more to train.                                                           |
   +----------+--------------------------------------------------------------------------+
   | Validate | Validate your model to obtain the metric you configured in the           |
   |          | ``searcher.metric`` field of your experiment config.                     |
   +----------+--------------------------------------------------------------------------+
   | Report   | Use the Core API to report results to the master.                        |
   +----------+--------------------------------------------------------------------------+

#. Create a ``3_hpsearch.py`` training script by copying the ``2_checkpoints.py`` script you created
   in :ref:`core-checkpoints`.

#. In your ``if __name__ == "__main__"`` block, access the hyperparameter values chosen for this
   trial using the ClusterInfo API and configure the training loop accordingly:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.py
      :language: python
      :dedent:
      :start-at: hparams = info.trial.hparams

#. Modify ``main()`` to run the train-validate-report loop mentioned above by iterating through
   ``core_context.searcher.operations()``. Each :class:`~determined.core.SearcherOperation` from
   :meth:`~determined.core.SearcherContext.operations` has a ``length`` attribute that specifies the
   absolute length of training to complete. After validating, report the searcher metric value using
   ``op.report_completed()``.

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.py
      :language: python
      :dedent:
      :start-at: batch = starting_batch
      :end-at: op.report_completed

#. Because the training length can vary, you might exit the train-validate-report loop before saving
   the last of your progress. To handle this, add a conditional save after the loop ends:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.py
      :language: python
      :dedent:
      :start-at: if last_checkpoint_batch != steps_completed
      :end-at: save_state

#. Create a new ``3_hpsearch.yaml`` file and add an ``entrypoint`` that invokes ``3_hpsearch.py``:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.yaml
      :language: yaml
      :lines: 1-2

   Add a ``hyperparameters`` section with the integer-type ``increment_by`` hyperparameter value
   that referenced in the training script:

   .. literalinclude:: ../../../../examples/tutorials/core_api/3_hpsearch.yaml
      :language: yaml
      :start-at: hyperparameters:
      :end-at: maxval

#. Run the code using the command:

   .. code:: bash

      det e create 3_hpsearch.yaml . -f

The complete ``3_hpsearch.py`` and ``3_hpsearch.yaml`` listings used in this example can be found in
the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.
