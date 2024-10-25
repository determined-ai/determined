:orphan:

**New Features**

-  Experiments: Add a ``name`` field to ``log_policies``. When a log policy matches, its name shows
   as a label in the WebUI, making it easy to spot specific issues during a run. Labels appear in
   both the run table and run detail views.

   In addition, there is a new format: ``name`` is required, and ``action`` is now a plain string.
   For more details, refer to :ref:`log_policies <config-log-policies>`.
