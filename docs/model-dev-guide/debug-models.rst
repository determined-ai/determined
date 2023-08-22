.. _model-debug:

#####################
 How to Debug Models
#####################

Using Determined to debug models depends on your environment.

Your code on a Determined cluster differs from typical training scripts in the following ways:

-  The code conforms to the Trial APIs as a subclass of the Determined ``Trial`` class, indirectly,
   by using one of the concrete subclasses, such as :class:`~determined.pytorch.PyTorchTrial`.
-  The code runs in a Docker container on another machine.
-  Your model can run many times in a hyperparameter search.
-  Your model can be distributed across multiple GPUs or machines.

These debugging steps introduce code changes incrementally, working toward a fully functioning
Determined model. Follow the nine steps as applicable to your environment:

-  Model-related Issues
      -  `Step 1 - Verify that your code runs locally`_
      -  `Step 2 - Verify that each Trial subclass method works locally`_
      -  `Step 3 - Verify local test mode`_

-  Docker- or Cluster-related Issues
      -  `Step 4 - Verify that the original code runs in a notebook or shell`_
      -  `Step 5 - Verify that each Trial subclass method works in a notebook or shell`_
      -  `Step 6 - Verify that local test mode works in a notebook or shell`_

-  Higher-level Issues
      -  `Step 7 - Verify that cluster test mode works with slots_per_trial set to 1`_
      -  `Step 8 - Verify that a single-GPU experiment works`_
      -  `Step 9 - Verify that a multi-GPU experiment works`_

**************
 Prerequisite
**************

Successful cluster installation as described in :ref:`Install the Determined cluster
<install-cluster>`.

.. _step1:

*********************************************
 Step 1 - Verify that your code runs locally
*********************************************

This step assumes you have ported (converted) your model from code outside of Determined. Otherwise,
skip to :ref:`Step 2 <step2>`.

Confirm that your code works as expected before continuing.

.. _step2:

***************************************************************
 Step 2 - Verify that each Trial subclass method works locally
***************************************************************

This step assumes you have a working local environment for training. If you typically run your code
in a Docker environment, skip to :ref:`Step 4 <step4>`. This step also ensures that your class
performs as expected by calling its methods and verifying the output.

#. Create simple tests to verify each ``Trial`` subclass method.

   Examples of what these tests might look like for :class:`~determined.pytorch.PyTorchTrial` and
   :class:`~determined.keras.TFKerasTrial` can be found in the
   :meth:`determined.TrialContext.from_config` documentation, but only you can verify what is
   reasonable for your test.

#. Diagnose failures:

   If you experience issues running the ``Trial`` subclass methods locally, it is likely there are
   errors are in your trial class or the ``hyperparameters`` section of your configuration file.
   Ideally, method-by-method evaluation makes it easier to find and solve issues.

.. _step3:

*********************************
 Step 3 - Verify local test mode
*********************************

:ref:`Step 2 <step2>` validated that your Trial API calls work as expected. This step uses your code
to run an actual Determined training loop with abbreviated workloads to make sure that it meets
Determined requirements.

This step assumes you have a working local environment for training. If you do not, skip to
:ref:`Step 4 <step4>`.

#. Create an experiment using the following command:

   .. code:: bash

      det experiment create myconfig.yaml my_model_dir --local --test

   The ``--local`` argument specifies that training occurs where you launched the experiment instead
   of occurring on a cluster. The ``--test`` argument runs abbreviated workloads to try to detect
   bugs sooner and exits immediately.

   The test is considered to have passed if the command completes successfully.

#. Diagnose failures:

   Local test mode performs the following actions:

   #. Builds a model.
   #. Runs a single batch of training data.
   #. Evaluates the model.
   #. Saves a checkpoint to a dummy location.

   If your per-method checks in :ref:`Step 2 <step2>` passed but local test mode fails, your
   ``Trial`` subclass might not be implemented correctly. Double-check the documentation. It is also
   possible that you have found a bug or an invalid assumption in the Determined software and should
   `file a GitHub issue <https://github.com/determined-ai/determined/issues/new>`__ or contact
   Determined on `Slack
   <https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew>`__.

.. _step4:

********************************************************************
 Step 4 - Verify that the original code runs in a notebook or shell
********************************************************************

This step is the same as :ref:`Step 1 <step1>`, except the original code runs on the Determined
cluster instead of locally.

#. Launch a notebook or shell on the cluster:

   Pass the root directory containing your model and training scripts in the ``--context`` argument:

   If you prefer a Jupyter notebook, enter:

   .. code:: bash

      det notebook start --context my_model_dir
      # Your browser should automatically open the notebook.

   If you prefer to use SSH to interact with your model, enter:

   .. code:: bash

      det shell start --context my_model_dir
      # Your terminal should automatically connect to the shell.

   Note that changes made to the ``--context`` directory while inside the notebook or shell do not
   affect the original files outside of the notebook or shell. See :ref:`notebook-state` for more
   information.

#. Verify code execution:

   After you are on the cluster, testing is the same as :ref:`Step 1 <step1>`.

#. Diagnose failures:

   -  If you are unable to start the container and receive a message about the context directory
      exceeding the maximum allowed size, it is because the ``--context`` directory cannot be larger
      than 95MB. If you need larger model definition files, consider setting up a bind mount using
      the ``bind_mounts`` field of the :ref:`task configuration <command-notebook-configuration>`.
      The :ref:`prepare-data` document lists additional strategies for accessing files inside a
      containerized environment.

   -  You might be referencing files that exist locally but are outside of the ``--context``
      directory. If the files are small, you may be able to copy them into the ``--context``
      directory. Otherwise, bind mounting the files can be an option.

   -  If you get dependency errors, dependencies might be installed locally that are not installed
      in the Docker environment used on the cluster. See :ref:`custom-env` and
      :ref:`custom-docker-images` for available options.

   -  If you need environment variables to be set for your model to work, see
      :ref:`command-notebook-configuration`.

.. _step5:

******************************************************************************
 Step 5 - Verify that each Trial subclass method works in a notebook or shell
******************************************************************************

This step is the same as :ref:`Step 2 <step2>`, except the original code runs on the Determined
cluster instead of locally.

#. Launch a notebook or shell:

   If you prefer to use Jupyter notebook, enter:

   .. code:: bash

      det notebook start --context my_model_dir
      # Your browser should automatically open the notebook.

   If you prefer to use SSH to interact with your model, enter:

   .. code:: bash

      det shell start --context my_model_dir
      # Your terminal should automatically connect to the shell.

   When interacting with the shell or notebook, testing is the same as :ref:`Step 2 <step2>`.

#. Diagnose failures:

   Combine the failure diagnosis steps used in :ref:`Step 2 <step2>` and :ref:`Step 4 <step4>`.

.. _step6:

*******************************************************************
 Step 6 - Verify that local test mode works in a notebook or shell
*******************************************************************

This step is the same as :ref:`Step 3 <step3>`, except the original code runs on the Determined
cluster instead of locally.

#. Launch a notebook or shell as described in :ref:`Step 4 <step4>`.

   On the cluster, testing is the same as :ref:`Step 3 <step3>`, except that the second model
   definition argument of the ``det experiment create`` command should be
   ``/run/determined/workdir`` or ``.`` if you have not changed the working directory after
   connecting to the cluster. This is because the ``--context`` specified when creating the shell or
   notebook is copied to the ``/run/determined/workdir`` directory inside the container, the same as
   the model definition argument is copied to ``det experiment create``.

#. Diagnose failures following the same steps described in :ref:`Step 3 <step3>` and :ref:`Step 4
   <step4>`.

.. _step7:

****************************************************************************
 Step 7 - Verify that cluster test mode works with slots_per_trial set to 1
****************************************************************************

This step is similar to :ref:`Step 6 <step6>`, except instead of launching the command from an
interactive environment, it is submitted to the cluster and managed by Determined.

#. Apply customizations:

   If you customized your command environment in testing :ref:`Step 3 <step3>`, :ref:`Step 4
   <step4>`, or :ref:`Step 5 <step5>`, make sure to apply the same customizations in your experiment
   configuration file.

#. Set ``resources.slots_per_trial``:

   Confirm that your experiment config does not specify ``resources.slots_per_trial`` or that it is
   set to ``1``. For example:

   .. code:: yaml

      resources:
        slots_per_trial: 1

#. Create an experiment with the ``--test`` argument, omitting the ``--local`` argument:

   .. code:: bash

      det experiment create myconfig.yaml my_model_dir --test

#. Diagnose failures:

   If you can run local test mode inside a notebook or shell but are unable to successfully submit
   an experiment, make sure that notebook or shell customizations you might have made are replicated
   in your :ref:`experiment configuration <experiment-config-reference>`, such as:

   -  If required, a custom Docker image is set in the experiment configuration.

   -  ``pip install`` or ``apt install`` commands needed in the interactive environment are built
      into a custom Docker image or included in the ``startup-hook.sh`` file in the model definition
      directory root. See :ref:`startup-hooks` for more information.

   -  Custom bind mounts required in the interactive environment are specified in the experiment
      configuration.

   -  Environment variables are correctly set in the experiment configuration.

   If no customizations are missing, the following new layers introduced with a cluster-managed
   experiment could be the cause of the problem:

   -  The ``checkpoint_storage`` settings are used for cluster-managed training. If
      ``checkpoint_storage`` is not configured in the experiment configuration or the master
      configuration, an error message can occur during experiment configuration validation before
      the experiment or trials are created. Correct this by providing a ``checkpoint_storage``
      configuration in one of the following locations:

      -  :ref:`master-config-reference`
      -  :ref:`experiment-config-reference`

   -  For a cluster-based experiment, configured ``checkpoint_storage`` settings are validated
      before training starts. The message ``Checkpoint storage validation failed``, indicates that
      you should review the ``checkpoint_storage`` setting values.

   -  The experiment configuration is more strictly validated for cluster-managed experiments than
      for ``--local --test`` mode. Errors related to ``invalid experiment configuration`` when
      attempting to submit the experiment to the cluster indicate that the experiment configuration
      has errors. Review the :ref:`experiment configuration <experiment-config-reference>`.

If you are unable to identify the cause of the problem, contact Determined `community support
<https://join.slack.com/t/determined-community/shared_invite/zt-cnj7802v-KcVbaUrIzQOwmkmY7gP0Ew>`__!

.. _step8:

****************************************************
 Step 8 - Verify that a single-GPU experiment works
****************************************************

This step is similar to :ref:`Step 7 <step7>`, except that it introduces hyperparameter search and
executes full training for each trial.

#. Configure your system the same as :ref:`Step 7 <step7>`:

   Confirm that your experiment configuration does not specify ``resources.slots_per_trial`` or that
   it is set to ``1``. For example:

   .. code:: yaml

      resources:
        slots_per_trial: 1

#. Create an experiment without the ``--test`` or ``--local`` arguments:

   You might find the ``--follow``, or ``-f``, argument helpful:

   .. code:: bash

      det experiment create myconfig.yaml my_model_dir -f

#. Diagnose failures:

   If :ref:`Step 7 <step7>` worked but this step does not, check:

   -  Check if the error happens when the experiment configuration has ``searcher.source_trial_id``
      set. One possibility in an actual experiment that does not occur in a ``--test`` experiment is
      the loading of a previous checkpoint. Errors when loading from a checkpoint can be caused by
      architectural changes, where the new model code is not architecturally compatible with the old
      model code.

   -  Generally, issues in this step are caused by doing training and evaluation continuously. Focus
      on how that change can cause issues with your code.

.. _step9:

***************************************************
 Step 9 - Verify that a multi-GPU experiment works
***************************************************

This step is similar to :ref:`Step 8 <step8>`, except that it introduces distributed training. This
step only applies if you have multiple GPUs and want to use distributed training.

#. Configure your system the same as :ref:`Step 7 <step7>`:

   Set ``resources.slots_per_trial`` to a number greater than ``1``. For example:

   .. code:: yaml

      resources:
        slots_per_trial: 2

#. Create your experiment:

   .. code:: bash

      det experiment create myconfig.yaml my_model_dir -f

#. Diagnose failures:

   If you are using the ``determined`` library APIs correctly, distributed training should work
   without error. Otherwise, common problems might be:

   -  If your experiment is not being scheduled on the cluster, ensure that the ``slots_per_trial``
      setting is valid for your cluster. For example:

      -  If you have four Determined agents running with four GPUs each, your ``slots_per_trial``
         could be ``1``, ``2``, ``3``, or ``4``, which fits on a single machine.
      -  A ``slots_per_trial`` value of ``8``, ``12``, or ``16`` completely utilizes a number of
         agent machines.
      -  A ``slots_per_trial`` value of ``5`` implies more than one agent but it is not a multiple
         of agent size so this is an invalid case.
      -  A ``slots_per_trial`` value of ``32`` is too large for the cluster and is also an invalid
         case.

      Ensure that there are no other notebooks, shells, or experiments on the cluster that might
      consume too many resources and prevent the experiment from starting.

   -  Determined is designed to control the details of distributed training for you. If you also try
      to control those details, such as by calling ``tf.config.set_visible_devices()`` in a
      :class:`~determined.keras.TFKerasTrial`, it is likely to cause issues.

   -  Some classes of metrics must be specially calculated during distributed training. Most
      metrics, such as loss or accuracy, can be calculated piecemeal on each worker in a distributed
      training job and averaged afterward. Those metrics are handled automatically by Determined and
      do not need special handling. Other metrics, such as F1 score, cannot be averaged from
      individual worker F1 scores. Determined has tooling for handling these metrics. See the
      documentation for using custom metric reducers with :ref:`PyTorch <pytorch-custom-reducers>`.
