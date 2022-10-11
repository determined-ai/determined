.. _core-metrics:

################
 Report Metrics
################

The Core API makes it easy to report training and validation metrics to the master during training
with only a few new lines of code.

#. For this example, create a new training script called ``1_metrics.py`` by copying the
   ``0_start.py`` script from :ref:`core-getting-started`.

#. Begin by importing import the ``determined`` module:

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :start-after: NEW: import determined
      :end-before: def main

#. Enable ``logging``, using the ``det.LOG_FORMAT`` for logs. This enables useful log messages from
   the ``determined`` library, and ``det.LOG_FORMAT`` enables filter-by-level in the WebUI.

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :start-at: logging.basicConfig
      :end-at: logging.error

#. In your ``if __name__ == "__main__"`` block, wrap the entire execution of ``main()`` within the
   scope of :meth:`determined.core.init`, which prepares resources for training and cleans them up
   afterward. Add the ``core_context`` as a new argument to ``main()`` because the Core API is
   accessed through the ``core_context`` object.

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :start-at: with det.core.init

#. Within ``main()``, add two calls: (1) report training metrics periodically during training and
   (2) report validation metrics every time a validation runs.

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.py
      :language: python
      :pyobject: main

   The ``report_validation_metrics()`` call typically happens after the validation step, however,
   actual validation is not demonstrated by this example.

#. Create a ``1_metrics.yaml`` file with an ``entrypoint`` invoking the new ``1_metrics.py`` file.
   You can copy the ``0_start.yaml`` configuration file and change the first couple of lines:

   .. literalinclude:: ../../../../examples/tutorials/core_api/1_metrics.yaml
      :language: yaml
      :lines: 1-2

#. Run the code using the command:

   .. code:: bash

      det e create 1_metrics.yaml . -f

#. You can now navigate to the new experiment in the WebUI and view the plot populated with the
   training and validation metrics.

The complete ``1_metrics.py`` and ``1_metrics.yaml`` listings used in this example can be found in
the :download:`core_api.tgz </examples/core_api.tgz>` download or in the `Github repository
<https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api>`_.
