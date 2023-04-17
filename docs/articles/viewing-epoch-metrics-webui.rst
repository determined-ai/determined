.. _coreapi-epoch-metrics-howto:

##############################################
 How to View Epoch-Based Metrics in the WebUI
##############################################

.. meta::
   :description: Learn how to view epoch-based metric data in the WebUI by reporting metrics to the Core API.
   :keywords: CoreAPI, WebUI, epoch-based, metrics, metric data

You can view epoch-based metric data in the WebUI by reporting an epoch metric to the Determined
master via the Core API. To do this, you'll need to define an epoch metric. This metric is used as
the ``x-axis`` label in the WebUI.

This article shows you how to view epoch-based metric data in the WebUI.

**Prerequisites**

-  :doc:`Quickstart for Model Developers <../quickstart-mdldev>`
-  A Determined cluster

**Recommended**

-  :doc:`Core API User Guide <../training/apis-howto/api-core-ug>`

****************************
 Step 1: Run the Experiment
****************************

In this article, we'll be training a PyTorch MNIST model using the Core API. Before reporting
metrics, we'll run our experiment.

.. note::

   To follow along with the steps shown in this article, you'll need to download the Core API
   PyTorch MNIST Tutorial files :download:`core_api_pytorch_mnist.tgz
   </examples/core_api_pytorch_mnist.tgz>`. You can also find the files by visiting the `Github
   repository
   <https://github.com/determined-ai/determined/tree/master/examples/tutorials/core_api_pytorch_mnist>`_.

For this step, we'll use our ``model_def.py`` script and its accompanying ``const.yaml`` experiment
configuration file.

From the directory containing our files, we'll run the following command:

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

For this section, we'll use our ``model_def_metrics.py`` script and its accompanying
``metrics.yaml`` experiment configuration file.

We'll start by importing Determined:

.. code:: python

   import determined as det

Step 2.1: Modify the Main Loop
==============================

We'll need a ``core.Context`` object for interacting with the master. To accomplish this, we'll
modify the __main__loop to include ``core_context``:

.. note::

   Refer to the ``if __name__ == "__main__":`` block in ``model_def_metrics.py``

.. literalinclude:: ../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: modify main loop core context
   :end-at: main(core_context=core_context)

Step 2.2: Modify the Train Method
=================================

Next, we'll use ``core_context.train`` to report training and validation metrics. We'll also modify
our code to report epoch-based metrics.

To begin, we'll modify the train() method by adding
``core_context.train.report_training_metrics()``:

.. literalinclude:: ../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: report training metrics
   :end-before: # Docs snippet end: report training metrics
   :dedent:

and ``core_context.train.report_validation_metrics()``:

.. literalinclude:: ../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: report validation metrics
   :end-before: # Docs snippet end: report validation metrics
   :dedent:

Since we've reported an epoch value, **Epoch** will be an available option for the X-Axis when we
view our metric data graph in the WebUI.

Step 2.3: Modify the Test Method
================================

Now, we'll modify the ``test()`` function header to include ``args`` and other elements we’ll need
during the evaluation loop. In addition, we'll pass the newly created ``core_context`` into both
``train()`` and ``test()``:

.. literalinclude:: ../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: pass core context
   :end-before: # Docs snippet end: pass core context
   :dedent:

We'll create a ``steps_completed`` variable to plot metrics on a graph in the WebUI:

.. literalinclude:: ../../examples/tutorials/core_api_pytorch_mnist/model_def_metrics.py
   :language: python
   :start-after: # Docs snippet start: calculate steps completed
   :end-before: # Docs snippet end: calculate steps completed
   :dedent:

Step 2.4: Run the Experiment
============================

To run our experiment, we'll run the following command:

.. code::

   det e create metrics.yaml .

Open the Determined WebUI again and navigate to the **Overview** tab.

The WebUI now displays metrics.

.. image:: ../assets/images/webui-metrics-epoch-based.png
   :width: 100%
   :alt: Epoch-based metrics in the WebUI

************
 Next Steps
************

In this article, you learned how to add a few lines of code to a script for the purpose of reporting
training and validation metrics to the Determined master via the Core API and viewing epoch-based
metric data in the WebUI.

You can visit the :doc:`/tutorials/index` to learn the basics of working with Determined and how to
port your existing code to the Determined environment.
