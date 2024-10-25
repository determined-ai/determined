:orphan:

**New Features**

-  Experiments: ``log_policies`` now have a ``name`` field. When a log policy matches, its name will
   be displayed as a label in the WebUI, allowing for easy identification of specific issues during
   a run. These labels will appear in both the run table and run detail views.

   It has a new format. ``name`` is required. ``pattern`` and ``action`` are optional. To make
   things simpler, user no longer needs to specify the ``type`` field to set an action. For example:

   Old format:

   .. code:: yaml

      log_policies:
        - pattern: ".*uncorrectable ECC error encountered.*"
          action:
            type: exclude_node
        - pattern: ".*CUDA out of memory.*"
            action:
              type: cancel_retries

   New format:

   .. code:: yaml

      log_policies:
        - name: ECC Error
          pattern: ".*uncorrectable ECC error encountered.*"
          action: exclude_node
        - name: CUDA OOM
          pattern: ".*CUDA out of memory.*"
          action: cancel_retries

   Both old and new format are supported at this time. We plan to deprecate the old format in the
   future.

   For more details, refer to :ref:`log_policies <config-log-policies>`.
