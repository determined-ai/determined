################
 Best Practices
################

***************************************
 General Tips for the Trial Definition
***************************************

Do:

-  Use framework abstractions to implement learning rate scheduling instead of directly changing the
   learning rate. See `tf.keras.optimizers.schedules.LearningRateSchedule
   <https://www.tensorflow.org/api_docs/python/tf/keras/optimizers/schedules/LearningRateSchedule>`__
   and :class:`determined.pytorch.LRScheduler` as examples.

-  For code that needs to download artifacts (e.g., data, configurations, pretrained weights),
   download to a `tempfile.TemporaryDirectory <https://docs.python.org/3/library/tempfile.html>`__
   unique to the Python process. This will avoid race conditions when using distributed training, in
   which Determined executes multiple Python processes in the same task container.

.. include:: ../../_shared/note-dtrain-learn-more.txt

Do not use instance attributes on a trial class to save any state over time (e.g., storing metric
history in a ``self`` attribute). The ``Trial`` instance will only save and restore model weights
and optimizer state over time; ``self`` attributes may be reset to their initial state at any time
if the Determined cluster reschedules the trial to another task container.

**********************************
 Separate Configuration from Code
**********************************

We encourage a clean separation of code from configuration via the :ref:`experiment configuration
<experiment-config-reference>`. Specifically, you are encouraged to use the pre-defined fields in
the experiment configuration, such as the ``searcher``, ``hyperparameters``, ``optimizations``, and
``resources``. This not only allows you to reuse the trial definition when you tune different
configuration fields but also improve the visualibility because those fields can be browsed in our
WebUI.

Do:

-  Move any hardcoded scalar values to the :ref:`hyperparameters
   <experiment-configuration_hyperparameters>` or :ref:`data <experiment-config-data>` fields in the
   experiment configuration. Use :func:`context.get_hparam() <determined.TrialContext.get_hparam>`
   or :func:`context.get_data_config() <determined.TrialContext.get_data_config>` to reference them
   in code.

-  Move any hardcoded filesystem paths (e.g., ``/data/train.csv``) to the ``data`` field of the
   experiment configuration. Use ``context.get_data_config()`` to reference them in code.

Do not use global variables in your model definition; consider moving them to the experiment
configuration.

*************************
 Understand Dependencies
*************************

We encourage tracking the dependencies associated with every workflow via the :ref:`environment
<experiment-config-reference>` field. Understanding and standardizing the environment you use to
execute Python in your development environment will pay off dividends in **portability**, allowing
you to flexibly move between local, cloud, and on-premise cluster environments.

Do:

-  Ramp up quickly by using our :ref:`default environment Docker image <default-environment>`,
   optionally specifying additional PyPI dependencies by using ``pip install`` in
   ``startup-hook.sh``.

-  As your dependencies increase in complexity, invest in :ref:`building and using a custom Docker
   image <custom-env>` that meets your needs.

-  Pin Python package dependencies to specific versions (e.g., ``<package>==<version>``) in build
   tools.

Do not modify the ``PYTHONPATH`` or ``PATH`` environment variables to import libraries by
circumventing the Python packaging system.
