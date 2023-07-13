:orphan:

**New Features**

-  API: A new patch API end point ``/api/v1/master/config`` has been added that allows the user to
   make changes to the master config while the cluster is still running. Currently, only changing
   the log config is supported.

-  CLI: A new CLI command has been added ``det master config --log --level <log_level> --color
   <on/off>`` that allows the user to change the log level and/or color of the master config while
   the cluster is still running. ``det master config`` can still be used to get master config.
