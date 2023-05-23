.. _deepspeed-autotuning:

####################
 DeepSpeed Autotune
####################

.. meta::
   :description: This user guide demonstrates how to optimize DeepSpeed parameter in order to take full advantage of the user's hardware and model.

Getting the most out of DeepSpeed (DS) requires aligning the many DS parameters with the specific
properties of your hardware and model. Determined AI's DeepSpeed Autotune (``dsat``) helps to
optimize these settings through an easy-to-use API with very few changes required in user-code, as
we describe in the remainder of this user guide. ``dsat`` can be used with any of
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial`, Core API, and HuggingFace Trainer.

****************
 What to Expect
****************

Assuming you have DeepSpeed code which already functions, autotuning is as easy as inserting one or
two helper functions into your code and then replacing the usual experiment launching command

.. code::

   det experiment create deepspeed.yaml .

with

.. code::

   python3 -m determined.pytorch.dsat asha deepspeed.yaml .

where ``deepspeed.yaml`` is the Determined configuration for a ``single`` experiment. There is no
need to write a special configuration file to use ``dsat``.

The above uses the ASHA algorithm (TODO: link to ASHA page) to tune the DS parameters and is one of
the three search methods we currently provide:

-  ``asha``: adaptively searches over randomly selected DeepSpeed configurations, yielding more
      compute resources to well-performing classes of configurations.
-  ``binary``: simple binary searches over the batch size for randomly-generated DS configurations.
-  ``random``: a search over random DeepSpeed configurations with an aggressive early-stopping
   criteria based on domain-knowledge of DeepSpeed and the search history.

DeepSpeed Autotune is built on top of Custom Searcher (TODO: link) which starts up two separate
experiments:

-  A ``single`` search runner experiment whose role is to coordinate and schedule the actual trials
   which run model code.
-  A ``custom`` experiment which contains the trials referenced above whose results are reported
   back to the search runner.

The logs of the ``single`` search runner experiment will contain information regarding the size of
the model, the GPU memory available, the activation memory required per example, and an approximate
computation of the maximum batch size per zero stage (which is used to guide the starting point for
all autotuning searches). When a best-performing DS configuration is found, the corresponding
``json`` configuration file will be written to the search runner's checkpoint directory, along with
a file detailing the configuration's corresponding metrics.

The ``custom`` experiment instead holds the visualization and summary tables which outline the
results for every Trial. Initially, a one-step profiling Trial is created to gather the above
information regarding the model and available hardware. Subsequently, multiple short Trials are
submitted which each report back metrics such as ``FLOPS_per_gpu``, ``throughput`` (samples/second),
and latency timing information.

In the following sections, we describe the specific user-code changes which must be made if using
``dsat`` with :class:`~determined.pytorch.deepspeed.DeepSpeedTrial`, Core API, and HuggingFace
Trainer, respectively.

********************
 ``DeepSpeedTrial``
********************

Determined's DeepSpeed Autotune works by writing DS configuration options to the
``overwrite_deepspeed_args`` field of the ``hyperparameters`` dictionary that is seen by each trial.
Assuming your default DS configuration is specified in a ``json`` file, then you only need to
incorporate these overwrite values into your original configuration in order to take advantage of
``dsat``.

In order to facilitate this process, we require that you add a ``deepspeed_config`` field under your
experiment's ``hyperparameters`` section which defines the relative path to the DS ``json``
configuration file. This is how ``dsat`` is informed of your default settings. For instance, if your
default DeepSpeed configuration is in ``ds_config.json`` which is placed at the top-level of your
model directory, then you would have:

.. code:: yaml

   hyperparameters:
     deepspeed_config: ds_config.json

The appropriate settings dictionary for each trial can then be easily accessed using the
``dsat.get_ds_config_from_hparams`` helper function, which can then be passed to
``deepspeed.initialize``, as usual:

.. code:: python

   from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext
   from determined.pytorch import dsat


   class MyDeepSpeedTrial(DeepSpeedTrial):
     def __init__(self, context: DeepSpeedTrialContext) -> None:
         self.hparams = self.context.get_hparams()
         config = dsat.get_ds_config_from_hparams(self.hparams)
         model = #...
         model_parameters= #...

         model_engine, optimizer, train_loader, lr_scheduler = deepspeed.initialize(
             model=model, model_parameters=parameters, config=config
         )

No further changes to user code are required to use DeepSpeed Autotune with a
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` instance.

**********
 Core API
**********

If using DeepSpeed Autotune with a Core API experiment, one additional change is needed after
following the steps in the ``DeepSpeedTrial`` section above: the ``forward``, ``backward``, and
``step`` methods of the ``DeepSpeedEngine`` class need to be wrapped in the
:func:`~dsat.dsat_reporting_context` context manager. This addition captures the autotuning metrics
from each trial and reports the results back to the Determined master.

A sketch of example ``dsat`` code with Core API:

.. code:: python

   for op in core_context.searcher.operations():
      for (inputs, labels) in trainloader:
          with dsat.dsat_reporting_context(core_context, op): # <-- The new code
              outputs = model_engine(inputs)
              loss = criterion(outputs, labels)
              model_engine.backward(loss)
              model_engine.step()

where ``core_context`` is the :class:`~determined.core.Context` instance which was initialized with
``determined.core.init``. The context manager requires access to both ``core_context`` and the
current :class:`~determined.core.SearcherOperation` instance (``op``) in order to appropriately
report results.

*********************
 HuggingFace Trainer
*********************

DeepSpeed Autotune can also be used with the HuggingFace (HF) Trainer and Determined AI's
:class:`~determined.integrations.huggingface.DetCallback` callback object.

As in the above cases, a ``deepspeed_config`` field specifying the relative path to the DS ``json``
config file must again be added to the ``hyperparameters`` section of the experiment configuration.
Reporting results back to the Determined master now requires both using the
``dsat.dsat_reporting_context`` context manager and the ``DetCallback`` callback object listed
above. Additionally, because ``dsat`` performs a search over different batch sizes and HuggingFace
expects parameters to be specified through command-line arguments, an additional helper is needed to
create consistent HuggingFace arguments: :func:`~dsat.get_hf_args_with_overwrites``.

The key pieces of relevant code from a HuggingFace Trainer script are below.

.. code:: python

   from determined.integrations.huggingface import DetCallback
   from determined.pytorch import dsat
   from transformers import HfArgumentParser,Trainer, TrainingArguments,

   hparams = self.context.get_hparams()
   parser = HfArgumentParser(TrainingArguments)
   args = sys.argv[1:]
   args = dsat.get_hf_args_with_overwrites(args, hparams)
   training_args = parser.parse_args_into_dataclasses(args, look_for_args_file=False)

   det_callback = DetCallback(core_context, ...)
   trainer = Trainer(args=training_args, ...)
   with dsat.dsat_reporting_context(core_context, op=det_callback.current_op):
       train_result = trainer.train(resume_from_checkpoint=checkpoint)

Things to note:

-  The ``dsat_reporting_context`` context manager shares the same initial
   :class:`~determined.core.SearcherOperation` as the ``DetCallback`` instance through its
   ``op=det_callback.current_op`` argument.

-  The entire ``train`` method of the HuggingFace trainer is now wrapped in the
   ``dsat_reporting_context`` context manager.

******************
 Advanced Options
******************

The command-line entrypoint to ``dsat`` has various available options, some of them
search-algorithm-specific. All available options for any given search method can be found as in

.. code::

   python3 -m determined.pytorch.dsat asha --help

and similar for the other search methods. Below, we highlight particularly important flags and
describe the search algorithms in some more detail.

General Options
===============

The following options are available for every search method.

By default, ``dsat`` launches 50 Trials and runs up to 16 concurrently. These values can be changed
via the ``--max-trials`` and ``--max-concurrent-trials`` flags. There is also an option to limit the
number of Trials by specifying ``--max-slots``. Other notable flags include:

-  ``--max-trials``: The maximum total number of trials to run. Default: 50.

-  ``--max-concurrent-trials``: The maximum total number of trials that can run concurrently.
   Default: 16.

-  ``--max-slots``: The maximum total number of slots that can run concurrently. Defaults to
   ``None``, i.e., there is no limit by default.

-  ``--metric``: specifies the metric to be optimized. Defaults to ``FLOPS-per-gpu``. Other
   available options are ``throughput``, ``forward``, ``backward``, and ``latency``.

-  ``--run-full-experiment``: When this flag is specified, after every ``dsat`` Trial has completed,
   a single-Trial experiment will be launched using the specifications in the ``deepspeed.yaml``
   overwritten with the best-found DS configuration parameters.

-  ``--zero-stages``: by default, ``dsat`` will search over each of stages ``1, 2, and 3``. This
   flag allows the user to limit the search to a subset of the stages by providing a space-separated
   list, as in ``--zero-stages 2 3``

``asha`` Options
================

The ``asha`` search algorithm randomly generates various DeepSpeed configurations and attempts to
tune the batch size for each such configuration through a binary search. ``asha`` adaptively
allocates resources to explore each configuration (providing more resources to promising lineages)
where the resource is the number of steps (i.e., launched trials) taken in each binary search.

``asha`` can be configured with the following flags:

-  ``--max-rungs``: The maximum total number of rungs to use in the ASHA algorithm. Larger values
   allow for longer binary searches. Default: 5.
-  ``--min-binary-search-trials``: The minimum number of trials to use for each binary search. The
   ``r`` parameter in `Link the ASHA paper <https://arxiv.org/abs/1810.05934>`. Default: 2.
-  ``--divisor``: Factor controlling the increase in The ``eta`` parameter in `Link the ASHA paper
   <https://arxiv.org/abs/1810.05934>`. Default: 2.
-  ``--asha-early-stopping``:
-  ``--search_range_factor``:

``binary`` Options
==================

``random`` Options
==================
