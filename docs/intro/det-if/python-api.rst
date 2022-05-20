.. _python-api:

############
 Python API
############

You can interact with a Determined cluster with the Python API.

The client module exposes many of the same capabilities as the det CLI tool directly to Python code with an object-oriented interface.

As a simple example, let’s walk through the most basic workflow for creating an experiment, waiting for it to complete, and finding the top-performing checkpoint.

The first step is to import the client module and possibly to call login():

.. code-block:: python

    from determined.experimental import client

    # We will assume that you have called `det user login`, so this is unnecessary:
    # client.login(master=..., user=..., password=...)

The next step is to call create_experiment():

.. code-block:: python

    # config can be a path to a config file or a python dict of the config.
    exp = client.create_experiment(config="my_config.yaml", model_dir=".")
    print(f"started experiment {exp.id}")

The returned object will be an ExperimentReference which has methods for controlling the lifetime of the experiment running on the cluster. In this example, we will just wait for the experiment to complete.

.. code-block:: python

    exit_status = exp.wait()
    print(f"experiment completed with status {exit_status}")

Now that the experiment has completed, you can grab the top-performing checkpoint from training:

.. code-block:: python

    best_checkpoint = exp.top_checkpoint()
    print(f"best checkpoint was {best_checkpoint.uuid}")
