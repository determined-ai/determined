.. _config-template:

#########################
 Configuration Templates
#########################

At a typical organization, many Determined configuration files will contain similar settings. For
example, all of the training workloads run at a given organization might use the same checkpoint
storage configuration. One way to reduce this redundancy is to use *configuration templates*. With
this feature, users can move settings that are shared by many experiments into a single YAML file
that can then be referenced by configurations that require those settings.

Each configuration template has a unique name and is stored by the Determined master. If a
configuration specifies a template, the effective configuration of the task will be the result of
merging the two YAML files (configuration file and template). The semantics of this merge operation
is described below. Determined stores this expanded configuration so that future changes to a
template will not affect the reproducibility of experiments that used a previous version of the
configuration template.

A single configuration file can use at most one configuration template. A configuration template
cannot itself use another configuration template.

***********************************
 Working with Templates in the CLI
***********************************

The :ref:`Determined command-line interface <install-cli>` can be used to list, create, update, and
delete configuration templates. This functionality can be accessed through the ``det template``
sub-command. This command can be abbreviated as ``det tpl``.

To list all the templates stored in Determined, use ``det template list``. You can also use the
``-d`` or ``--detail`` option to show additional details.

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

*******************************************************
 Using Templates to Simplify Experiment Configurations
*******************************************************

An experiment can use a configuration template by using the ``--template`` command-line option to
specify the name of the desired template.

Here is an example demonstrating how an experiment configuration can be split into a reusable
template and a simplified configuration.

Consider the experiment configuration below:

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

You may find that the values for the ``checkpoint_storage`` field are the same for many experiments
and you want to use a configuration template to reduce the redundancy. You might write a template
like the following:

.. code:: yaml

   description: template-tf-gpu
   checkpoint_storage:
     type: s3
     access_key: my-access-key
     secret_key: my-secret-key
     bucket: my-bucket-name

Then the experiment configuration for this experiment can be written as below:

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

****************
 Merge Behavior
****************

Suppose we have a template that specifies top-level fields ``a`` and ``b`` and a configuration that
specifies fields ``b`` and ``c``. The merged configuration will have fields ``a``, ``b``, and ``c``.
The value for field ``a`` will simply be the value set in the template. Likewise, the value for
field ``c`` will be whatever was specified in the configuration. The final value for field ``b``,
however, depends on the value's type:

-  If the field specifies a scalar value, the merged value will be the one specified by the
   configuration (the configuration overrides the template).

-  If the field specifies a list value, the merged value will be the concatenation of the list
   specified in the template and that specified in the configuration.

   Note that there are exceptions to this rule for ``bind_mounts`` and ``resources.devices``. It may
   be the case that the both the original config and the template will attempt to mount to the same
   ``container_path``, which would result in an unsable config. In those situations, the original
   config is preferred, and the conflicting bind mount or device from the template is omittied in
   the merged result.

-  If the field specifies an object value, the resulting value will be the object generated by
   recursively applying this merging algorithm to both objects.
