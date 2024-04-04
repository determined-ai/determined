:orphan:

**New Features**

-  CLI: Add commands, ``det e set log-retention <exp-id>`` and ``det t set log-retention
   <trial-id>`` to allow the user to set the log retention days for experiments and trials. These
   commands take the required argument ``--days`` and the optional argument ``forever``. ``--days``
   sets the number of days to retain the logs for, from the time of creation and ``--forever``
   retains logs forever. The allowed range for ``--days`` is between ``-1`` and ``32767``, where -1
   retains logs forever.

-  WebUI: Add support for retaining logs for multiple experiments by selecting experiments from the
   Experiment List page and choosing **Retain Logs** from **Actions**. Users can then input the
   desired number of days for log retention or select the "Forever" checkbox for indefinite log
   retention. The allowed range for the number of days is between ``-1`` and ``32767``, where -1
   retains logs forever. Add a new column, ''Log Retention Days'', to the Trial List page. ''Log
   Retention Days'' displays the duration logs will be retained for each trial from creation.
