:orphan:

**New Features**

-  Logging: Add a configuration option, ``logging_retention``, to the server configuration file with
   options for scheduling (``schedule``) log cleanup, and for selecting the number of ``days`` to
   retain logs. Experiments can override the default log retention settings provided by the server
   by specifying ``log_retention_days`` in the experiment configuration. Valid values for retention
   days range from ``-1`` to ``32767``, and schedules must adhere to a valid cron expression or
   duration format. If retention days is set to ``-1``, logs will be retained indefinitely.
   Conversely, setting retention days to 0 will result in logs being deleted during the next
   scheduled log cleanup. Additionally, administrators can manually initiate log retention cleanup
   using the ``det master cleanup-logs command``.
