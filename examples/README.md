# Determined Examples

## Tutorials

| Example                                                       | Dataset          | Framework             |
|:-------------------------------------------------------------:|:----------------:|:---------------------:|
| [mnist\_pytorch](tutorials/mnist_pytorch)                     | MNIST            | PyTorch               |
| [fashion\_mnist\_tf\_keras](tutorials/fashion_mnist_tf_keras) | Fashion MNIST    | TensorFlow (tf.keras) |
| [imagenet\_pytorch](tutorials/imagenet_pytorch)               | ImageNet PyTorch | PyTorch               |
| [core\_api](tutorials/core_api)                               | Core API         | -                     |

## Computer Vision

| Example                                                                      | Dataset                      | Framework                                |
|:----------------------------------------------------------------------------:|:----------------------------:|:----------------------------------------:|
| [cifar10\_pytorch](computer_vision/cifar10_pytorch)                          | CIFAR-10                     | PyTorch                                  |
| [cifar10\_pytorch\_inference](computer_vision/cifar10_pytorch_inference)     | CIFAR-10                     | PyTorch                                  |
| [fasterrcnn\_coco\_pytorch](computer_vision/fasterrcnn_coco_pytorch)         | Penn-Fudan Dataset           | PyTorch                                  |
| [mmdetection\_pytorch](computer_vision/mmdetection_pytorch)                  | COCO                         | PyTorch                                  |
| [detr\_coco\_pytorch](computer_vision/detr_coco_pytorch)                     | COCO                         | PyTorch                                  |
| [deformabledetr\_coco\_pytorch](computer_vision/deformabledetr_coco_pytorch) | COCO                         | PyTorch                                  |
| [cifar10\_tf\_keras](computer_vision/cifar10_tf_keras)                       | CIFAR-10                     | TensorFlow (tf.keras)                    |
| [iris\_tf\_keras](computer_vision/iris_tf_keras)                             | Iris Dataset                 | TensorFlow (tf.keras)                    |
| [unets\_tf\_keras](computer_vision/unets_tf_keras)                           | Oxford-IIIT Pet Dataset      | TensorFlow (tf.keras)                    |
| [efficientdet\_pytorch](computer_vision/efficientdet_pytorch)                | COCO                         | PyTorch                                  |
| [byol\_pytorch](computer_vision/byol_pytorch)                                | CIFAR-10 / STL-10 / ImageNet | PyTorch                                  |
| [deepspeed\_cifar10_cpu_offloading](deepspeed/cifar10_cpu_offloading)        | CIFAR-10                     | PyTorch (DeepSpeed)                      |

## Natural Language Processing (NLP)

| Example                                            | Dataset    | Framework |
|:--------------------------------------------------:|:----------:|:---------:|
| [albert\_squad\_pytorch](nlp/albert_squad_pytorch) | SQuAD      | PyTorch   |
| [bert\_glue\_pytorch](nlp/bert_glue_pytorch)       | GLUE       | PyTorch   |
| [word\_language\_model](nlp/word_language_model)   | WikiText-2 | PyTorch   |

## HP Search Benchmarks

| Example                                                                         | Dataset               | Framework |
|:-------------------------------------------------------------------------------:|:---------------------:|:---------:|
| [darts\_cifar10\_pytorch](hp_search_benchmarks/darts_cifar10_pytorch)           | CIFAR-10              | PyTorch   |
| [darts\_penntreebank\_pytorch](hp_search_benchmarks/darts_penntreebank_pytorch) | Penn Treebank Dataset | PyTorch   |

## Neural Architecture Search (NAS)

| Example                            | Dataset | Framework |
|:---------------------------------:|:-------:|:---------:|
| [gaea\_pytorch](nas/gaea_pytorch) | DARTS   | PyTorch   |

## Meta Learning

| Example                                                                | Dataset  | Framework |
|:----------------------------------------------------------------------:|:--------:|:---------:|
| [protonet\_omniglot\_pytorch](meta_learning/protonet_omniglot_pytorch) | Omniglot | PyTorch   |

## Diffusion

|                                        Example                                        |  Dataset  | Framework |
|:-------------------------------------------------------------------------------------:|:---------:|:---------:|
| [textual\_inversion\_stable\_diffusion](diffusion/textual_inversion_stable_diffusion) | det_logos |  PyTorch  |

## Generative Adversarial Networks (GAN)

| Example                                       | Dataset          | Framework             |
|:---------------------------------------------:|:----------------:|:---------------------:|
| [dc\_gan\_tf\_keras](gan/dcgan_tf_keras)      | MNIST            | TensorFlow (tf.keras) |
| [gan\_mnist\_pytorch](gan/gan_mnist_pytorch)  | MNIST            | PyTorch               |
| [deepspeed\_dcgan](deepspeed/deepspeed_dcgan) | MNIST / CIFAR-10 | PyTorch (DeepSpeed)   |
| [pix2pix\_tf\_keras](gan/pix2pix_tf_keras)    | pix2pix          | TensorFlow (tf.keras) |

## Custom Reducers

| Example                                                                    | Dataset | Framework  |
|:--------------------------------------------------------------------------:|:-------:|:----------:|
| [custom\_reducers\_mnist\_pytorch](features/custom_reducers_mnist_pytorch) | MNIST   | PyTorch    |

## HP Search Constraints

| Example                                                                  | Dataset | Framework  |
|:------------------------------------------------------------------------:|:-------:|:----------:|
| [hp\_constraints\_mnist\_pytorch](features/hp_constraints_mnist_pytorch) | MNIST   | PyTorch    |

## Custom Search Method

| Example                                                                  | Dataset | Framework  |
|:------------------------------------------------------------------------:|:-------:|:----------:|
| [asha\_search\_method](custom_search_method/asha_search_method)          | MNIST   | PyTorch    |
