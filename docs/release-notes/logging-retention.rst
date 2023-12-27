:orphan:

**New Features**

-  Logging: Log retention can now be configured in the server config through the
   ``logging_retention`` section, with optional ``days`` and ``schedule`` entries. Experiments can
   override the server provided default by setting ``log_retention_days`` in the experiment config.
   Values of ``-1`` to ``32767`` are valid for retention days, and schedules must be a valid cron
   expression or duration. If retention days is set to ``-1``, logs will be retained indefinitely.
   If retention days is set to ``0``, logs will will be deleted on the next scheduled log cleanup.
   Log retention cleanup can be run manually by admins with the ``det master cleanup-logs`` command.
