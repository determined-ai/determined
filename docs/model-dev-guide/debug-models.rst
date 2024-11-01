.. _model-debug:

##################
 Debugging Models
##################

Using Determined to train your model can introduce a number of failure points that aren't present
when running training scripts locally. Running your code on a Determined cluster differs from
typical training scripts in the following ways:

-  The code runs in a Docker container, possibly on another machine.
-  Your model can run many times in a hyperparameter search.
-  Your model can be distributed across multiple GPUs or machines.

These debugging steps introduce environment and code changes incrementally, working toward fully
functioning distributed training on a Determined cluster:

-  `Step 1 - Verify that your training script runs locally`_
-  `Step 2 - Verify that your training script runs in a notebook or shell`_
-  `Step 3 - Verify that a single-GPU experiment works`_
-  `Step 4 - Verify that a multi-GPU experiment works`_

**************
 Prerequisite
**************

Ensure you have successfully installed a Determined cluster. The cluster can be installed on a local
development machine, on-prem, or on cloud. For installation guides, visit :ref:`installation-guide`.

.. _step1:

********************************************************
 Step 1 - Verify that your training script runs locally
********************************************************

Determined's training APIs are designed to work both on-cluster and locally (that is, without
interacting with a Determined master), so you should be able to run your training script on the same
local machine that you ran your model before integrating with Determined APIs.

If you ported your model to :class:`~determined.pytorch.PyTorchTrial` or
:class:`~determined.pytorch.deepspeed.DeepSpeedTrial` and are having trouble getting your ported
model to work, one debugging strategy is to manually call the various methods of your Trial directly
before calling ``Trainer.fit()``.

Confirm that your code works as expected before continuing.

.. _step2:

***********************************************************************
 Step 2 - Verify that your training script runs in a notebook or shell
***********************************************************************

This step is the same as :ref:`Step 1 <step1>`, except the your training script runs on the
Determined cluster instead of locally.

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

   After you are on the cluster, you can test your script by just running it, as in :ref:`Step 1
   <step1>`.

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

.. _step3:

****************************************************
 Step 3 - Verify that a single-GPU experiment works
****************************************************

In this step, instead of launching the command from an interactive environment, it is submitted to
the cluster and managed by Determined.

#. Apply customizations:

   If you customized your command environment in testing :ref:`Step 2 <step2>`, make sure to apply
   the same customizations in your experiment configuration file.

#. Set ``entrypoint``:

   Set the ``entrypoint`` of your experiment config to match the way you call your training script
   in your environment, including all arguments.

#. Set ``resources.slots_per_trial``:

   Confirm that your experiment config does not specify ``resources.slots_per_trial`` or that it is
   set to ``1``. For example:

   .. code:: yaml

      resources:
        slots_per_trial: 1

#. Submit your experiment:

   .. code:: bash

      det experiment create myconfig.yaml my_model_dir -f

#. Diagnose failures:

   The experiment configuration is validated upon submission. If you see errors about ``invalid
   experiment configuration`` during submission, review the :ref:`experiment configuration
   <experiment-config-reference>`.

   If your training script runs inside a notebook or shell, but fails when on-cluster, make sure
   that notebook or shell customizations you might have made are replicated in your experiment
   config, such as:

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

If you are unable to identify the cause of the problem, contact Determined `community support
<https://determined-community.slack.com/join/shared_invite/zt-1f4hj60z5-JMHb~wSr2xksLZVBN61g_Q>`__!

.. _step4:

***************************************************
 Step 4 - Verify that a multi-GPU experiment works
***************************************************

This step introduces distributed training.

#. Make any necessary code changes:

   -  If you are using the Core API for training, distributed training may take extra work. The
      :ref:`api-core-ug-basic` and :ref:`api-core-ug` examples can help you understand what all is
      required.

   -  If you are using Determined's :class:`keras.DeterminedCallback
      <determined.keras.DeterminedCallback>` for training, you will have to take the `standard steps
      for enabling distributed training in Keras
      <https://www.tensorflow.org/tutorials/distribute/keras>`__, except that you don't need to
      configure the ``TF_CONFIG`` environment variable because it is configured by Determined's
      :ref:`launch-tensorflow`.

   -  For the remaining training APIs, distributed training should work without additional code
      changes.

#. Wrap your training script in ``entrypoint`` with the correct launcher for the training API you
   are using. For example, if you are using PyTorchTrial, you should use Determined's
   :ref:`pytorch-dist-launcher`:

   .. code:: yaml

      entrypoint: >-
        python3 -m determined.launch.torch_distributed --
        python3 ./my_train_script.py --my-option=value

   See :ref:`predefined-launchers` for more launcher options.

#. Configure ``resources.slots_per_trial`` to a number greater than ``1``. For example:

   .. code:: yaml

      resources:
        slots_per_trial: 2

#. Submit your experiment:

   .. code:: bash

      det experiment create myconfig.yaml my_model_dir -f

#. Diagnose failures:

   Double-check that any code changes you made are correct, and also that you wrapped your code with
   the correct launcher. Otherwise, common problems might be:

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

   -  Determined is designed to control many of the details of distributed training for you. If you
      also try to control those details, such as by calling ``tf.config.set_visible_devices()``
      while training a Keras model, it is likely to cause issues.

   -  Some classes of metrics must be specially calculated during distributed training. Most
      metrics, such as loss or accuracy, can be calculated piecemeal on each worker in a distributed
      training job and averaged afterward. Those metrics are handled automatically by Determined and
      do not need special handling. Other metrics, such as F1 score, cannot be averaged from
      individual worker F1 scores. Determined has tooling for handling these metrics. See the
      documentation for using custom metric reducers with :ref:`PyTorch <pytorch-custom-reducers>`.
