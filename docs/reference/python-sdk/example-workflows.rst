.. _python-sdk-examples:

##############################
 Python SDK Example Workflows
##############################

Walk through how to use the Python SDK in these basic and advanced workflow examples.

.. tabs::

   .. tab::

      Basic Workflow

      **Find the Top Performing Checkpoint**

      In this example, we'll walk through the most basic workflow for creating an experiment,
      waiting for it to complete, and finding the top-performing checkpoint.

      The first step is to import the client module and possibly to call login():

      .. code:: python

         from determined.experimental import client

         # We will assume that you have called `det user login`, so this is unnecessary:
         # client.login(master=..., user=..., password=...)

      The next step is to call create_experiment():

      .. code:: python

         # Config can be a path to a config file or a Python dict of the config.
         exp = client.create_experiment(config="my_config.yaml", model_dir=".")
         print(f"started experiment {exp.id}")

      The returned object is an ``Experiment`` object, which offers methods to manage the
      experiment's lifecycle. In the following example, we simply await the experiment's completion.

      .. code:: python

         exit_status = exp.wait()
         print(f"experiment completed with status {exit_status}")

      Now that the experiment has completed, you can grab the top-performing checkpoint from
      training:

      .. code:: python

         best_checkpoint = exp.list_checkpoints()[0]
         print(f"best checkpoint was {best_checkpoint.uuid}")

   .. tab::

      Advanced Workflow

      **Run and Administer Experiments**

      Visit the `det-python-sdk-demo
      <https://github.com/determined-ai/determined-examples/tree/e499000d92a0a973d1f40a419934f393957a3296/blog/python_sdk_demo>`__
      to learn how to run and administer experiments using the Python SDK.
