.. _example-solutions:

##########
 Examples
##########

Start with an example machine learning model converted to Determined's APIs. Code examples are in
the ``examples/`` subdirectory of the `Determined GitHub repo
<https://github.com/determined-ai/determined/tree/master/examples>`__. Download links are below.

For more examples, visit the `determined-examples repo
<https://github.com/determined-ai/determined-examples/>`__.

Each example includes a model definition and one or more experiment configuration files. To run an
example, download the appropriate ``.tgz`` file, extract it, ``cd`` into the directory, and use
``det experiment create`` to create a new experiment, passing in the appropriate configuration file.
For example, here is how to train the ``mnist_pytorch`` example with a fixed set of hyperparameters:

.. code::

   tar xzvf mnist_pytorch.tgz
   cd mnist_pytorch
   det experiment create const.yaml .

For an introduction to using the training APIs, please visit :ref:`Training APIs
<apis-howto-overview>`.

*****************
 Computer Vision
*****************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  MNIST
      -  :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>`

***********
 DeepSpeed
***********

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  DeepSpeed (PyTorch)
      -  Enron Email Corpus
      -  :download:`gpt_neox.tgz </examples/gpt_neox.tgz>`

********************
 DeepSpeed Autotune
********************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  DeepSpeed (PyTorch)
      -  ImageNet (Generated)
      -  :download:`torchvision.tgz </examples/torchvision.tgz>`

   -  -  HuggingFace (DeepSpeed/PyTorch)
      -  Beans (HuggingFace)
      -  :download:`hf_image_classification.tgz </examples/hf_image_classification.tgz>`

   -  -  HuggingFace (DeepSpeed/PyTorch)
      -  WikiText (HuggingFace)
      -  :download:`hf_language_modeling.tgz </examples/hf_language_modeling.tgz>`

***********
 Diffusion
***********

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  det_logos
      -  :download:`textual_inversion_stable_diffusion.tgz
         </examples/textual_inversion_stable_diffusion.tgz>`
