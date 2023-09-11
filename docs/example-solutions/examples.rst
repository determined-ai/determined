.. _example-solutions:

##########
 Examples
##########

Get started quickly by using an example machine learning model that has been converted to
Determined's APIs. Visit the ``examples/`` subdirectory of the `Determined GitHub repo
<https://github.com/determined-ai/determined/tree/master/examples>`__ or download the link below.

Each example includes a model definition and one or more experiment configuration files. To run an
example, download the appropriate ``.tgz`` file, extract it, ``cd`` into the directory, and use
``det experiment create`` to create a new experiment, passing in the appropriate configuration file.
For example, here is how to train the ``mnist_pytorch`` example with a fixed set of hyperparameters:

.. code::

   tar xzvf mnist_pytorch.tgz
   cd mnist_pytorch
   det experiment create const.yaml .

For an introduction to using the training API, please visit the Training APIs section.

*****************
 Computer Vision
*****************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  CIFAR-10
      -  :download:`cifar10_pytorch.tgz </examples/cifar10_pytorch.tgz>`

   -  -  PyTorch
      -  MNIST
      -  :download:`mnist_pytorch.tgz </examples/mnist_pytorch.tgz>`

   -  -  PyTorch
      -  ImageNet
      -  :download:`imagenet_pytorch.tgz </examples/imagenet_pytorch.tgz>`

   -  -  PyTorch
      -  Penn-Fudan Dataset
      -  :download:`fasterrcnn_coco_pytorch.tgz </examples/fasterrcnn_coco_pytorch.tgz>`

   -  -  PyTorch
      -  COCO
      -  :download:`detr_coco_pytorch.tgz </examples/detr_coco_pytorch.tgz>`

   -  -  PyTorch
      -  COCO
      -  :download:`deformabledetr_coco_pytorch.tgz </examples/deformabledetr_coco_pytorch.tgz>`

   -  -  PyTorch
      -  COCO
      -  :download:`efficientdet_pytorch.tgz </examples/efficientdet_pytorch.tgz>`

   -  -  PyTorch
      -  COCO
      -  :download:`detectron2_coco_pytorch.tgz </examples/detectron2_coco_pytorch.tgz>`

   -  -  TensorFlow (tf.keras)
      -  CIFAR-10
      -  :download:`cifar10_tf_keras.tgz </examples/cifar10_tf_keras.tgz>`

   -  -  TensorFlow (tf.keras)
      -  Iris Dataset
      -  :download:`iris_tf_keras.tgz </examples/iris_tf_keras.tgz>`

   -  -  TensorFlow (tf.keras)
      -  Oxford-IIIT Pet Dataset
      -  :download:`unets_tf_keras.tgz </examples/unets_tf_keras.tgz>`

   -  -  PyTorch
      -  CIFAR-10 / STL-10 / ImageNet
      -  :download:`byol_pytorch.tgz </examples/byol_pytorch.tgz>`

***********************************
 Natural Language Processing (NLP)
***********************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  SQuAD 2.0
      -  :download:`albert_squad_pytorch.tgz </examples/albert_squad_pytorch.tgz>`

   -  -  PyTorch
      -  GLUE
      -  :download:`bert_glue_pytorch.tgz </examples/bert_glue_pytorch.tgz>`

   -  -  PyTorch
      -  WikiText-2
      -  :download:`word_language_model.tgz </examples/word_language_model.tgz>`

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

   -  -  DeepSpeed (PyTorch)
      -  CIFAR-10
      -  :download:`cifar10_moe.tgz </examples/cifar10_moe.tgz>`

   -  -  DeepSpeed (PyTorch)
      -  CIFAR-10
      -  :download:`pipeline_parallelism.tgz </examples/pipeline_parallelism.tgz>`

   -  -  DeepSpeed (PyTorch)
      -  MNIST / CIFAR-10
      -  :download:`deepspeed_dcgan.tgz </examples/deepspeed_dcgan.tgz>`

   -  -  DeepSpeed (PyTorch)
      -  CIFAR-10
      -  :download:`cifar10_cpu_offloading.tgz </examples/cifar10_cpu_offloading.tgz>`

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

************************
 HP Search Benchmarking
************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  CIFAR-10
      -  :download:`darts_cifar10_pytorch.tgz </examples/darts_cifar10_pytorch.tgz>`

   -  -  PyTorch
      -  Penn Treebank Dataset
      -  :download:`darts_penntreebank_pytorch.tgz </examples/darts_penntreebank_pytorch.tgz>`

**********************************
 Neural Architecture Search (NAS)
**********************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  DARTS
      -  :download:`gaea_pytorch.tgz </examples/gaea_pytorch.tgz>`

***************
 Meta Learning
***************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  Omniglot
      -  :download:`protonet_omniglot_pytorch.tgz </examples/protonet_omniglot_pytorch.tgz>`

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

****************************************
 Generative Adversarial Networks (GANs)
****************************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  MNIST
      -  :download:`gan_mnist_pytorch.tgz </examples/gan_mnist_pytorch.tgz>`

   -  -  TensorFlow (tf.keras)
      -  MNIST
      -  :download:`dcgan_tf_keras.tgz </examples/dcgan_tf_keras.tgz>`

   -  -  TensorFlow (tf.keras)
      -  pix2pix
      -  :download:`pix2pix_tf_keras.tgz </examples/pix2pix_tf_keras.tgz>`

********
 Graphs
********

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  PROTEINS
      -  :download:`proteins_pytorch_geometric.tgz </examples/proteins_pytorch_geometric.tgz>`

***************************
 Features: Custom Reducers
***************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  MNIST
      -  :download:`custom_reducers_mnist_pytorch.tgz </examples/custom_reducers_mnist_pytorch.tgz>`

*********************************
 Features: HP Search Constraints
*********************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  MNIST
      -  :download:`hp_constraints_mnist_pytorch.tgz </examples/hp_constraints_mnist_pytorch.tgz>`

********************************
 Features: Custom Search Method
********************************

.. list-table::
   :header-rows: 1

   -  -  Framework
      -  Dataset
      -  Filename

   -  -  PyTorch
      -  MNIST
      -  :download:`asha_search_method.tgz </examples/asha_search_method.tgz>`
