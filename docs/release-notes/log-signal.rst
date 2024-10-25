:orphan:

**New Features**

-  Experiments: ``log_policies`` now have a ``name`` field. When a log policy matches, its name will
   be displayed as a label in the WebUI, allowing for easy identification of specific issues during
   a run. These labels will appear in both the run table and run detail views.

   It has a new format. ``name`` is required, and ``action`` should be a plain string. For more
   details, refer to :ref:`log_policies <config-log-policies>`.
