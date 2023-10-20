.. _simple-metrics-reporting:

##################################
 Perform Simple Metrics Reporting
##################################

In this tutorial, we'll walk through how to perform simple metrics reporting using :ref:`detached
mode <detached-mode-index>`.

For the full script, visit the `Github repository
<https://github.com/determined-ai/determined/blob/main/examples/features/unmanaged/1_singleton.py>`_.

************
 Objectives
************

These step-by-step instructions walk you through the following tasks:

-  Setting up your training environment
-  Importing and initializing the core context
-  Setting the master address and executing your training script

Upon completing this user guide, you will:

-  Grasp the concept and application of detached mode
-  Successfully report metrics in detached mode
-  Navigate and visualize trials using the Determined WebUI

**********************************
 Set Up Your Training Environment
**********************************

To begin, you'll need a Determined cluster. If you are new to Determined AI (Determined), you can
install the Determined library and start a cluster locally.

-  Ensure you have Docker running and then run the following command:

.. code::

   pip install determined

   # If your machine has GPUs:
   det deploy local cluster-up

   # If your machine does not have GPUs:
   det deploy local cluster-up --no-gpu

.. include:: ../../_shared/note-pip-install-determined.txt

********************************************
 Step 1: Import and Initialize the Core API
********************************************

#. Start by importing the necessary modules for your training code:

.. code:: python

   import random
   from determined.experimental import core_v2

2. Initialize the core context to recognize the trial with some identifying metadata in the main
   function:

.. code:: python

   def main():
       core_v2.init(
           defaults=core_v2.DefaultConfig(
               name="detached_mode_example",
           ),
       )

3. Report your trial and validation metrics:

.. code:: python

   for i in range(100):
       core_v2.train.report_training_metrics(steps_completed=i, metrics={"loss": random.random()})
       if (i + 1) % 10 == 0:
           loss = random.random()
           print(f"validation loss is: {loss}")
           core_v2.train.report_validation_metrics(
               steps_completed=i, metrics={"loss": loss}
           )
   core_v2.close()

   if __name__ == "__main__":
       main()

*************************************************************
 Step 2: Set Master Address and Execute Your Training Script
*************************************************************

#. Define the Determined master address:

.. code:: bash

   export DET_MASTER=<DET_MASTER_IP:PORT>

2. Run your training script:

.. code:: bash

   python3 <my_training_script.py>

3. Visualize the metrics! Navigate to ``<DET_MASTER_IP:PORT>`` in your web browser to see the trial.

************
 Next Steps
************

You've now grasped the essence of simple metrics reporting in detached mode. This guide facilitated
understanding detached mode, running trials, and visualizing metrics. Dive deeper into other
features of Determined to maximize your model training efficiency!
