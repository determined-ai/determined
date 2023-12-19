.. _qs-webui:

##################
 WebUI Quickstart
##################

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

      #. Download and extract the tar file: :download:`mnist_pytorch.tgz
         <../examples/mnist_pytorch.tgz>`.

      #. Install the Determined library and start a cluster locally:

         .. code:: bash

            pip install determined
            det deploy local cluster-up

         If your local machine does not have a supported NVIDIA GPU, include the ``no-gpu`` option:

         .. code:: bash

            pip install determined
            det deploy local cluster-up --no-gpu

      #. In the ``mnist_pytorch`` directory, create an experiment specifying the ``const.yaml``
         configuration file:

         .. code:: bash

            det experiment create const.yaml .

         The final dot (.) argument uploads all of the files in the current directory as the
         *context directory* for your model. Determined copies the model context directory contents
         to the trial container working directory.

      #. In the WebUI dashboard, click the **Experiment** name to view the experiment’s trial
         display.

   .. tab::

      remotely

      Run a remote distributed training job.

      .. note::

         To run a remote distributed training job, you'll need a Determined cluster with multiple
         GPUs. In distributed training, A cluster is made up of a master and one or more agents. The
         master provides centralized management of the agent resources.

      #. Download and extract the tar file: :download:`mnist_pytorch.tgz
         <../examples/mnist_pytorch.tgz>`.

         If you examine the ``distributed.yaml`` file, you'll notice the
         ``resources.slots_per_trial`` field is set to a value of ``8``:

         .. code:: yaml

            resources:
               slots_per_trial: 8

         This is the number of available GPU resources. The ``slots_per_trial`` value must be
         divisible by the number of GPUs per machine. You can change the value to match your
         hardware configuration.

      #. To connect to a Determined master running on a remote instance, set the remote IP address
         and port number in the ``DET_MASTER`` environment variable:

         .. code:: bash

            export DET_MASTER=<ipAddress>:8080

      #. Create and run the experiment:

         .. code:: bash

            det experiment create distributed.yaml .

         You can also use the ``-m`` option to specify a remote master IP address:

         .. code:: bash

            det -m http://<ipAddress>:8080 experiment create distributed.yaml .

      #. In the WebUI dashboard, click the **Experiment** name to view the experiment’s trial
         display. The loss curve is similar to the single-GPU experiment in the previous exercise
         but the time to complete the trial is reduced by about half.
