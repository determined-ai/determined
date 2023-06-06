.. _config-template:

#########################
 Configuration Templates
#########################

In a typical organization, many Determined configuration files will share similar settings. This can
cause redundancy. For example, all training workloads run at a given organization might use the same
checkpoint storage configuration. One way to reduce this redundancy is to use *configuration
templates*. This feature allows users to consolidate settings shared across many experiments into a
single YAML file that can be referenced by configurations needings those settings.

Each configuration template has a unique name and is stored by the Determined master. If a
configuration employs a template, the effective configuration of the task will be the outcome of
merging the two YAML files (the configuration file and the template). The semantics of this merge
operation are described below. Determined stores this effective configuration to ensure future
changes to a template do not affect the reproducibility of experiments that used a previous version
of the configuration template.

A single configuration file can use at most one configuration template. A configuration template
cannot itself use another configuration template.

************************************************************
 Leveraging Templates to Simplify Experiment Configurations
************************************************************

An experiment can adopt a configuration template by using the ``--template`` command-line option to
denote the name of the desired template.

The following example demonstrates splitting an experiment configuration into a reusable template
and a simplified configuration.

.. code:: yaml

   name: mnist_tf_const
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name
   data:
     base_url: https://s3-us-west-2.amazonaws.com/determined-ai-datasets/mnist/
     training_data: train-images-idx3-ubyte.gz
     training_labels: train-labels-idx1-ubyte.gz
     validation_set_size: 10000
   hyperparameters:
     base_learning_rate: 0.001
     weight_cost: 0.0001
     global_batch_size: 64
     n_filters1: 40
     n_filters2: 40
   searcher:
     name: single
     metric: error
     max_length:
       batches: 500
     smaller_is_better: true

You may find that many experiments share the same values for the ``checkpoint_storage`` field,
leading to redundancy. To reduce the redundancy you could use a configuration template. For example,
consider the following template:

.. code:: yaml

   description: template-tf-gpu
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name

The experiment configuration for this experiment can then be written using the following code:

.. code:: yaml

   description: mnist_tf_const
   data:
     base_url: https://s3-us-west-2.amazonaws.com/determined-ai-datasets/mnist/
     training_data: train-images-idx3-ubyte.gz
     training_labels: train-labels-idx1-ubyte.gz
     validation_set_size: 10000
   hyperparameters:
     base_learning_rate: 0.001
     weight_cost: 0.0001
     global_batch_size: 64
     n_filters1: 40
     n_filters2: 40
   searcher:
     name: single
     metric: error
     max_length:
       batches: 500
     smaller_is_better: true

To launch the experiment with the template:

.. code:: bash

   $ det experiment create --template template-tf-gpu mnist_tf_const.yaml <model_code>

************************************
 Managing Templates through the CLI
************************************

The :ref:`Determined command-line interface <cli-ug>` provides tools for managing configuration
templates including listing, creating, updating, and deleting templates. This functionality can be
accessed through the ``det template`` sub-command. This command can be abbreviated as ``det tpl``.

To list all the templates stored in Determined, use ``det template list``. To show additional
details, use the ``-d`` or ``--detail`` option.

.. code::

   $ det tpl list
   Name
   -------------------------
   template-s3-tf-gpu
   template-s3-pytorch-gpu
   template-s3-keras-gpu

To create or update a template, use ``det tpl set template_name template_file``.

.. code::

   $ cat > template-s3-keras-gpu.yaml << EOL
   description: template-s3-keras-gpu
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name
   EOL
   $ det tpl set template-s3-keras-gpu template-s3-keras-gpu.yaml
   Set template template-s3-keras-gpu

****************
 Merge Behavior
****************

To demonstrate merge behavior when merging a template and a configuration, let's say we have a
template that specifies top-level fields ``a`` and ``b``, and a configuration that specifies fields
``b`` and ``c``. The resulting merged configuration will have fields ``a``, ``b``, and ``c``. The
value for field ``a`` will simply be the value set in the template. Likewise, the value for field
``c`` will be whatever was specified in the configuration. The final value for field ``b``, however,
depends on the value's type:

-  If the field specifies a scalar value, the configuration's value will take precedence in the
   merged configuration (overriding the template's value).

-  If the field specifies a list value, the merged value will be the concatenation of the list
   specified in the template and the one specified in the configuration.

   .. note::

      There are certain exceptions for ``bind_mounts`` and ``resources.devices``. There could be
      situations where both the original config and the template will attempt to mount to the same
      ``container_path``, resulting in an unstable configuration. In such scenarios, the original
      configuration is preferred, and the conflicting bind mount or device from the template is
      omitted in the merged result.

-  If the field specifies an object value, the resulting value will be the object generated by
   recursively applying this merging algorithm to both objects.
