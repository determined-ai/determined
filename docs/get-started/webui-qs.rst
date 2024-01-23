.. _qs-webui:

##############################
 Create Your First Experiment
##############################

If you already have a Determined cluster and are signed in to the WebUI, you can follow these steps
to see how to run your first experiment.

***************
 Prerequisites
***************

-  Your system must meet the software and hardware requirements described in the :ref:`System
   Requirements <system-requirements>`.
-  You must have a running Determined cluster. To set one up locally, visit :ref:`Basic Setup
   <basic>`.

*******************
 Run an Experiment
*******************

.. tabs::

   .. tab::

      locally

      Train a single model for a fixed number of batches, using constant values for all
      hyperparameters on a single *slot*. A slot is a CPU or CPU computing device, which the
      Determined master schedules to run.

      .. note::

         To run an experiment in a local training environment, your Determined cluster requires only
         a single CPU or GPU. A cluster is made up of a master and one or more agents. A single
         machine can serve as both a master and an agent.

      #. Download and extract the tar file: :download:`mnist_pytorch.tgz
         <../examples/mnist_pytorch.tgz>`.

      #. Open a terminal window and ``cd`` into the ``mnist_pytorch`` directory.

         The ``const.yaml`` file is a YAML-formatted :ref:`experiment configuration
         <experiment-config-reference>` file that corresponds to an example experiment.

      #. Create an experiment that specifies the ``const.yaml`` configuration file by typing the
         following :ref:`CLI <cli-ug>` command.

         .. code:: bash

            det experiment create const.yaml .

         The final dot (.) argument uploads all of the files in the current directory as the
         *context directory* for your model. Determined copies the model context directory contents
         to the trial container working directory.

      #. To view the experiment in your browser:

         -  Enter the following URL: **http://localhost:8080/**. This is the cluster address for
            your local training environment.
         -  Accept the default username of ``determined``, and click **Sign In**. A password is not
            required.

      #. Navigate to the home page and then visit your **Uncategorized** experiments.

         .. image:: /assets/images/qswebui-recent-local.png
            :alt: Determined AI WebUI Dashboard showing a user's recent experiment submissions

      #. Select the experiment to display the experiment’s details such as Metrics.

         .. image:: /assets/images/qswebui-metrics-local.png
            :alt: Determined AI WebUI Dashboard showing details for a local experiment

   .. tab::

      remotely

      Run a remote distributed training job.

      .. note::

         To run a remote distributed training job, you'll need a Determined cluster with multiple
         GPUs. In distributed training, A cluster is made up of a master and one or more agents. The
         master provides centralized management of the agent resources. By default, the
         :ref:`slots-per-trial` value is set to ``1`` which disables distributed training.

      #. Download and extract the tar file: :download:`mnist_pytorch.tgz
         <../examples/mnist_pytorch.tgz>`.

      #. Open a terminal window and ``cd`` into the ``mnist_pytorch`` directory.

      #. Using your code editor, examine the ``distributed.yaml`` file. Notice the
         ``resources.slots_per_trial`` field is set to a value of ``8``:

         .. code:: yaml

            resources:
               slots_per_trial: 8

         This is the number of available GPU resources. The ``slots_per_trial`` value must be
         divisible by the number of GPUs per machine.

         -  If necessary, use your code editor to change the value to match your hardware
            configuration.

      #. To connect to the Determined master running on your remote instance, set the remote IP
         address and port number in the ``DET_MASTER`` environment variable:

         .. code:: bash

            export DET_MASTER=<ipAddress>:8080

      #. Create and run the experiment:

         .. code:: bash

            det -u <username> experiment create distributed.yaml .

      #. To view the experiment in your browser:

         -  Enter the URL of your remote instance: **http://<ipAddress>:8080/**.
         -  Sign in using your username and password.

      #. Navigate to the home page and then visit **Your Recent Submissions**.

         .. image:: /assets/images/qswebui-recent-remote.png
            :alt: Determined AI WebUI Dashboard showing a user's recent experiment submissions

      #. Select the experiment to display the experiment’s details such as Metrics. Notice the loss
         curve is similar to the locally-run, single-GPU experiment but the time to complete the
         trial is reduced by about half.

         .. image:: /assets/images/qswebui-metrics-remote.png
            :alt: Determined AI WebUI Dashboard showing details for a remote distributed experiment

************
 Learn More
************

**Want to learn how to adapt your existing model code to Determined?**

The behavior of an experiment is configured via an experiment configuration, or YAML, file. A
configuration file is typically passed as a command-line argument when an experiment is created with
the :ref:`CLI <cli-ug>`.

-  Visit the :ref:`experiment-config-reference` for a complete description of the experiment
   configuration file.
-  Visit the :ref:`api-core-ug` for a walk-through of how to adapt your existing model code to
   Determined using the PyTorch MNIST model.
