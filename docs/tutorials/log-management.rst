.. _log-management:

#################
 Log Management
#################

This guide covers two log management features: Log Search and Log Signal.

*************
 Log Search
*************

To perform a log search:

1. Navigate to your run in the WebUI.
2. In the Logs tab, start typing in the search box to open the search pane.
3. To use regex search, click the "Regex" checkbox in the search pane.
4. Click on a search result to view it in context, with logs before and after visible.
5. Scroll up and down to fetch new logs.

Note: Search results are not auto-updating. You may need to refresh to see new logs.

***********
 Log Signal
***********

Log Signal allows you to configure log policies in the master configuration to display labels in the UI when specific patterns are matched in the logs.

To set up a log policy:

1. In the master configuration file, under ``task_container_defaults > log_policies``, define your log policies.
2. Each policy can have a ``name``, ``pattern``, and ``action``.
3. When a log matching the pattern is encountered, the ``name`` will be displayed as a label in the run table and run detail views.

Example configuration:

.. code:: yaml

   log_policies:
      - name: "CUDA OOM"
        pattern: ".*CUDA out of memory.*"
        action:
          type: cancel_retries

This will display a "CUDA OOM" label in the UI when a CUDA out of memory error is encountered in the logs.

For more detailed information on configuring log policies, refer to the :ref:`experiment configuration reference <config-log-policies>`.

