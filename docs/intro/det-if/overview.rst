#####################
Determined Interfaces
#####################

This section describes the mechanisms available for interacting with a cluster.

*****************
 WebUI Interface
*****************

Typically, you need to use the WebUI and the CLI.

The WebUI allows users to create and monitor the progress of experiments. It is accessible by
visiting ``http://master-addr:8080``, where ``master-addr`` is the hostname or IP address where the
Determined master is running.

***********************
 Command Line Interface
***********************

The :ref:`command-line interface (CLI) <cli>` is distributed as a Python wheel package. After the
wheel is installed, use the CLI ``det`` command to interact with the cluster.

**********
Python API
**********

TBD

*****************
Jupyter Notebooks
*****************

`Jupyter Notebooks <https://jupyter.org/>`__ are a convenient way to develop and debug machine
learning models, visualize the behavior of trained models, or even manage the training lifecycle of
a model manually. Determined makes it easy to launch and manage notebooks.

Determined Notebooks have the following benefits:

-  Jupyter Notebooks run in containerized environments on the cluster. We can easily manage
   dependencies using images and virtual environments. The HTTP requests are passed through the
   master proxy from and to the container.

-  Jupyter Notebooks are automatically terminated if they are idle for a configurable duration to
   release resources. A notebook instance is considered to be idle if it is not receiving any HTTP
   traffic and it is not otherwise active (as defined by the ``notebook_idle_type`` option in the
   :ref:`task configuration <command-notebook-configuration>`).

After a Notebook is terminated, it is not possible to restore the files that are not stored in the
persistent directories. **You need to ensure that the cluster is configured to mount persistent
directories into the container and save files in the persistent directories in the container.**
See :ref:`notebook-state` for more information.

If you open a Notebook tab in JupyterLab, it will automatically open a kernel that will not be
shut down automatically so you need to manually terminate the kernels.

*****************
 TensorBoards
*****************

`TensorBoard <https://www.tensorflow.org/tensorboard>`__ is a widely used tool for visualizing and
inspecting deep learning models. Determined makes it easy to use TensorBoard to examine a single
experiment or to compare multiple experiments.

.. _command-notebook-configuration:

*******************************
 Interactive Job Configuration
*******************************

The behavior of interactive jobs, such as :ref:`TensorBoards <tensorboards>`, :ref:`notebooks
<notebooks>`, :ref:`commands, and shells <commands-and-shells>`, can be influenced by setting a
variety of configuration variables. These configuration variables are similar but not identical to
the configuration options supported by :ref:`experiments <experiment-config-reference>`.

Configuration settings can be specified by passing a YAML configuration file when launching the
workload via the Determined CLI:

.. code::

   $ det tensorboard start experiment_id --config-file=my_config.yaml
   $ det notebook start --config-file=my_config.yaml
   $ det cmd run --config-file=my_config.yaml ...
   $ det shell start --config-file=my_config.yaml

Configuration variables can also be set directly on the command line when any Determined task,
except a TensorBoard, is launched:

.. code::

   $ det notebook start --config resources.slots=2
   $ det cmd run --config description="determined_command" ...
   $ det shell start --config resources.priority=1

Options set via ``--config`` take precedence over values specified in the configuration file.
Configuration settings are compatible with any Determined task unless otherwise specified.


.. toctree::
   :maxdepth: 1
   :hidden:

   webui-if
   commands-and-shells
   python-api
   notebooks
   tensorboard
