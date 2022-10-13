.. _core-getting-started:

#################
 Getting Started
#################

As a simple introduction, this example training script increments a single integer in a loop,
instead of training a model with machine learning. The changes shown for the example model should be
similar to the changes you make in your actual model.

The ``0_start.py`` training script used in this example contains your simple "model":

.. literalinclude:: ../../../../examples/tutorials/core_api/0_start.py
   :language: python
   :start-at: import

To run this script, create a configuration file with at least the following values:

.. literalinclude:: ../../../../examples/tutorials/core_api/0_start.yaml
   :language: yaml

The actual configuration file can have any name, but this example uses ``0_start.yaml``.

Run the code using the command:

.. code:: bash

   det e create 0_start.yaml . -f

If you navigate to this experiment in the WebUI no metrics are displayed because you have not yet
reported them to the master using the Core API.
