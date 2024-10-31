:orphan:

**Breaking Changes**

-  ASHA: All experiments using ASHA hyperparameter search must now configure ``max_time`` and
   ``time_metric`` in the experiment config, instead of ``max_length``. Additionally, training code
   must report the configured ``time_metric`` in validation metrics. As a convenience, Determined
   training loops now automatically report ``batches`` and ``epochs`` with metrics, which you can
   use as your ``time_metric``. ASHA experiments without this modification will no longer run.

-  Custom Searchers: all custom searchers including DeepSpeed Autotune were deprecated in ``0.36.0``
   and are now being removed. Users are encouraged to use a preset searcher, which can be easily
   :ref:`configured <experiment-configuration_searcher>` for any experiment.

**New Features**

-  API: introduce ``keras.DeterminedCallback``, a new high-level training API for TF Keras that
   integrates Keras training code with Determined through a single :ref:`Keras Callback
   <api-keras-ug>`.

-  API: introduce ``deepspeed.Trainer``, a new high-level training API for DeepSpeedTrial that
   allows for Python-side training loop configurations and includes support for local training.

**Deprecations**

-  Experiment Config: the ``max_length`` field of the searcher configuration section has been
   deprecated for all experiments and searchers. Users are expected to configure the desired
   training length directly in training code.

-  Experiment Config: the ``optimizations`` config has been deprecated. Please see :ref:`Training
   APIs <apis-howto-overview>` to configure supported optimizations through training code directly.

-  Experiment Config: the ``scheduling_unit`` config field has been deprecated. min
   checkpoint/val/records per epoch

-  Experiment Config: the ``entrypoint`` field no longer accepts ``model_def:TrialClass`` as trial
   definitions. Please invoke your training script directly (``python3 train.py``).

-  Core API: the ``SearcherContext`` (``core.searcher``) has been deprecated. Training code no
   longer requires ``core.searcher.operations`` to run, and progress should be reported through
   ``core.train.report_progress``.

-  Horovod: the horovod distributed training backend has been deprecated. Users are encouraged to
   migrate to the native distributed backend of their training framework (``torch.distributed`` or
   ``tf.distribute``).

-  Trial APIs: ``TFKerasTrial`` has been deprecated. Users are encouraged to migrate to the new
   :ref:`Keras Callback <api-keras-ug>`.

-  Launchers: the ``--trial`` argument in Determined launchers has been deprecated. Please invoke
   your training script directly.

-  ASHA: the ``stop_once`` field of the ``searcher`` config for ASHA searchers has been deprecated.
   All ASHA searches are now early-stopping based (``stop_once: true``) instead of promotion based.

-  CLI: The ``--test`` and ``--local`` flags for ``det experiment create`` have been deprecated. All
   training APIs now support local execution (``python3 train.py``). Please see ``training apis``
   for details specific to your framework.

-  Web UI: previously, trials that reported an ``epoch`` metric enabled an epoch X-axis in the Web
   UI metrics tab. This metric name has been changed to ``epochs``, with ``epoch`` as a fallback
   option.

**Removed Features**

-  WebUI: "Continue Training" no longer supports configurable number of batches in the Web UI and
   will simply resume the trial from the last checkpoint.
