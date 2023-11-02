:orphan:

**New Features**

-  Experiments: Add a new experiment config option ``log_policies`` to allow configuring policies to
   take after a regex is matched. This can also be configured at the cluster level or per resource
   pool through task container defaults. Please see :ref:`experiment-config-reference` and
   :ref:`master-config-reference` for more information.

   There are two action types a trial can be configured to take

   -  ``exclude_node``: If a trial fails and restarts, the trial will not schedule, for its
      restarts, on a node that reported a log that matched the regex provided. This can be used to
      allow trials to avoid being retried on nodes with certain hardware issues like uncorrectable
      gpu ECC errors.

   -  ``cancel_retries``: If a trial reports a log that matches this pattern, the trial will not be
      restarted. This is useful for certain errors that are not transient, such as a CUDA
      out-of-memory error caused by the model being too large for the allocated hardware.
