:orphan:

**New Features**

-  Experiments: Add a ``log_policies`` configuration option to define actions when a trial's log
   matches specified patterns.

   -  The ``exclude_node`` action prevents a failed trial's restart attempts (due to its
      max_restarts policy) from being scheduled on nodes with matched error logs. This is useful for
      bypassing nodes with hardware issues like uncorrectable GPU ECC errors.

   -  The ``cancel_retries`` action prevents a trial from restarting if a trial reports a log that
      matches the pattern, even if it has remaining max_restarts. This avoids using resources for
      retrying a trial that encounters certain failures that won't be fixed by retrying the trial,
      such as CUDA memory issues. For details, visit :ref:`experiment-config-reference` and
      :ref:`master-config-reference`.

This option is also configurable at the cluster or resource pool level via task container defaults.
