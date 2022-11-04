.. _core-distributed:

######################
 Distributed Training
######################

The Core API has special considerations for running distributed training. Some of the more important
considerations are:

-  Access to all IP addresses of every node in the Trial (through the ClusterInfo API).

-  Communication primitives such as :meth:`~determined.core.DistributedContext.allgather`,
   :meth:`~determined.core.DistributedContext.gather`, and
   :meth:`~determined.core.DistributedContext.broadcast` to give you out-of-the-box coordination
   between workers.

-  Since many distributed training frameworks expect all workers in training to operate in-step, the
   :meth:`~determined.core.PreemptContext.should_preempt` call is automatically synchronized across
   workers so that all workers decide to preempt or continue as a unit.

#. Create a ``4_distributed.py`` training script by copying the ``3_hpsearch.py`` from
   :ref:`core-hpsearch`.

#. Add launcher logic to execute one worker subprocess per slot.

   Start with a ``launcher_main()`` function that executes one worker subprocess per slot.

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :pyobject: launcher_main

   Typically, you do not have to write your own launcher. Determined provides launchers for Horovod,
   ``torch.distributed``, and DeepSpeed. Additionally, there are third-party launchers available,
   such as ``mpirun``. When using a custom or third-party launcher, wrap your worker script in the
   ``python -m determined.launcher.wrap_rank`` wrapper script so the WebUI log viewer can filter
   logs by rank.

   Also add a ``worker_main()`` that will run training on each slot:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :pyobject: worker_main

   Then modify your ``if __name__ == "__main__"`` block to invoke the correct ``*_main()`` based on
   command-line arguments:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: slots_per_node = len(info.slot_ids)

#. In the training code, use the ``allgather`` primitive to do a "distributed" increment, to gain
   experience using the communication primitives:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: all_increment_bys =
      :end-at: x += sum(all_increment_bys)

#. Usually, trial logs are easier to read when status is only printed on the chief worker:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-after: some logs are easier to read
      :end-at: logging.info

#. Only the chief worker is permitted to report training metrics, report validation metrics, upload
   checkpoints, or report searcher operations completed. This rule applies to the steps you take
   periodically during training:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: if steps_completed % 10 == 0
      :end-at: return

   The rule also applies to the steps you take after validating:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-after: only the chief may report validation metrics
      :end-at: op.report_completed

   The rule also applies to the conditional save after the main loop completes:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.py
      :language: python
      :dedent:
      :start-at: again, only the chief may upload checkpoints
      :end-at: save_state

#. Create a ``4_distributed.yaml`` file by copying the ``3_distributed.yaml`` file and changing the
   first couple of lines:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.yaml
      :language: yaml
      :lines: 1-2

   Set the ``resources.slots_per_trial`` field to the number of GPUs you want:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.yaml
      :language: yaml
      :start-at: resources:
      :end-at: slots_per_trial:

   You can return to using the ``single`` searcher instead of an ``adaptive_asha`` hyperparameter
   search:

   .. literalinclude:: ../../../../examples/tutorials/core_api/4_distributed.yaml
      :language: yaml
      :start-at: searcher:
      :end-at: max_length:

#. Run the code using the Determined CLI with the following command:

   .. code:: bash

      det e create 4_distributed.yaml . -f

The complete ``4_distributed.py`` and ``3_hpsearch.yaml`` listings used in this example can be found
in the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.
