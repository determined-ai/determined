.. _command-notebook-configuration:

#############################################
 Interactive Job Configuration
#############################################

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
