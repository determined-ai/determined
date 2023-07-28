.. _coreapi-epoch-metrics-howto:

##############################################
 How to View Epoch-Based Metrics in the WebUI
##############################################

.. meta::
   :description: Learn how to analyze and visualize training progress and validation performance over multiple epochs using the Core API.
   :keywords: CoreAPI, WebUI, epochs, metrics, metric data

Sometimes, you want to analyze and visualize your model's training progress and validation
performance over multiple epochs.

In this article, we'll show you how to view epoch-based metric data in the WebUI by reporting an
epoch metric to the Determined master via the Core API. To do this, we'll define an epoch metric.
This metric is then used as the X-Axis label in the WebUI.

**Recommended**

-  :doc:`Quickstart for Model Developers <../tutorials/quickstart-mdldev>`
-  :doc:`Core API User Guide <../model-dev-guide/apis-howto/api-core-ug>`

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

.. include:: ../_shared/note-pip-install-determined.txt

****************************
 Step 1: Run the Experiment
****************************

Before reporting metrics, we'll run our experiment and ensure we can see it in the WebUI.

.. note::

   To follow along with the steps shown in this article, you'll need to download the Core API
   PyTorch MNIST Tutorial files :download:`core_api_pytorch_mnist.tgz
   </examples/core_api_pytorch_mnist.tgz>`. You can also find the files by visiting the `Github
   repository
   <https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api_pytorch_mnist>`_.

For this step, we'll use our ``model_def.py`` script and its accompanying ``const.yaml`` experiment
configuration file.

From the directory containing our files, we'll begin by running the following command:

.. code:: bash

   det e create const.yaml . -f

We don't have any data to plot yet, but we'll visit the Determined WebUI to see that our experiment
is running.

To do this, we'll navigate to ``http://localhost:8080/``.

To sign in, we'll accept the default determined username, leave the password empty, and click **Sign
In**.

In the WebUI, we'll select our experiment, and then navigate to the **Logs** tab.

************************
 Step 2: Report Metrics
************************

In this section, we'll define our epoch metric. We'll also report training and validation metrics to
the Determined master. To do this, we'll import Determined, and create a
:class:`~determined.core.Context` object to allow interaction with the master. Then, we'll pass the
``core_context`` as an argument into ``main()``, ``train()``, and ``test()`` and modify the function
headers accordingly.

To follow along, use the ``model_def_metrics.py`` script and its accompanying ``metrics.yaml``
experiment configuration file.

We'll start by importing Determined:

.. code:: bash

   python3

.. code:: python

   import determined as det

Step 2.1: Verify the Main Loop Contains ``core.Context``
========================================================

We'll need a ``core.Context`` object for interacting with the master. Verify the __main__loop in
your script, ``model_def_metrics.py``, contains ``core_context``:

.. code:: python

   if __name__ == "__main__":

     # NEW: Establish new determined.core.Context and pass to main
     # function.
     with det.core.init() as core_context:
         main(core_context=core_context)

Step 2.2: Modify the Train and Validation Methods to Report Epoch-Based Metrics
===============================================================================

Our script already contains ``core_context.train``. This is used to report training and validation
metrics.

But, we also want to report epoch-based metrics. To do this, we'll modify the train() method to
include ``epoch_idx`` as a metric. This allows Determined to keep track of the specific epoch for
which training loss is being reported:

.. code:: python

   # NEW: Report epoch-based training metrics to Determined
   # master via core_context.
   # Index by (batch_idx + 1) * (epoch-1) * len(train_loader)
   # to continuously plot loss on one graph for consecutive
   # epochs.
   core_context.train.report_training_metrics(
       steps_completed=batches_completed + epoch_idx * len(train_loader),
       metrics={"train_loss": loss.item(), "epoch": epoch_idx},

   )

Similarly, we'll include ``epoch`` as a metric in the reported validation metrics. This allows
Determined to track the specific epoch for which the validation loss is being reported:

.. code:: python

   # NEW: Report epoch_based validation metrics to Determined master
   # via core_context.
   core_context.train.report_validation_metrics(
       steps_completed=steps_completed,
       metrics={"test_loss": test_loss, "epoch": epoch},

   )

Now that we've reported an epoch value, **Epoch** will be an available option for the X-Axis when we
view our metric data graph in the WebUI.

Step 2.3: Verify the Test Method
================================

We'll need a ``test()`` function to evaluate the trained model on the test/validation data for the
current epoch.

Verify your code contains a ``test()`` function header that includes ``args`` and other elements
needed during the evaluation loop. This function header should pass ``core_context`` into both
``train()`` and ``test()``:

.. code:: python

   # NEW: Pass args, test_loader, epoch, and steps_completed into
   # test().
   test(
       args,
       model,
       device,
       test_loader,
       epoch_idx,
       core_context,
       steps_completed=steps_completed,
   )
   scheduler.step()

In addition, we'll need a ``steps_completed`` variable to plot metrics on a graph in the WebUI. The
goal is to to track the progress of training by considering both the number of completed training
batches and the current epoch index. This allows Determined to continuously plot the training loss
on one graph for consecutive epochs.

Verify your code contains a ``steps_completed`` variable:

.. code:: python

   core_context.train.report_training_metrics(
      steps_completed=batches_completed + epoch_idx * len(train_loader),
      metrics={"train_loss": loss.item(), "epoch": epoch_idx),
   )

Step 2.4: Run the Experiment
============================

To run our experiment, we'll run the following command:

.. code::

   det e create metrics.yaml .

To view epoch-based metrics:

-  Open the Determined WebUI and select your experiment.

Your experiment opens in the **Overview** tab.

-  Select the **Metrics** tab.
-  Select the **X-Axis** menu and then choose **Epoch**.
-  Scroll down to view the epoch-based metrics graph.

.. image:: ../assets/images/webui-metrics-epoch-based.png
   :width: 100%
   :alt: Epoch-based metrics in the WebUI

*********
 Summary
*********

In this article, you learned how to add a few lines of code to a script for the purpose of reporting
epoch-based metrics in addition to training and validation metrics. You also learned how to view
epoch-based metric data in the WebUI.

************
 Next Steps
************

Now you can try editing your own script for the purpose of reporting epoch-based metrics to the
Determined master.

For more tutorials, visit the :doc:`/tutorials/index` to learn the basics of working with Determined
and how to port your existing code to the Determined environment.
