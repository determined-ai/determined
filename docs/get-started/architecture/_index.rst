.. _architecture-index:

:weight: 500

.. meta::
   :description: Learn how to quickly get started with Determined.

######################
 How Determined Works
######################

With Determined you can:

-  Use state-of-the-art distributed training to train models faster without changing model code.
-  Automatically find high-quality models using advanced hyperparameter tuning.
-  Get more from your GPUs and reduce cloud GPU costs with preemptible instances and smart
   scheduling.
-  Leverage experiment tracking out-of-the-box to track and reproduce your work, tracking code
   versions, metrics, checkpoints, and hyperparameters.
-  Continue using popular deep learning libraries, such as TensorFlow, Keras, and PyTorch by simply
   integrating the Determined API with your existing model code.

Determined integrates these features into an easy-to-use, high-performance deep learning environment
so you can spend your time building models instead of managing infrastructure.

.. image:: /assets/images/_det-ai-sys-arch-01-start-dark.png
   :class: only-dark
   :alt: Determined AI system architecture diagram dark mode

.. image:: /assets/images/_det-ai-sys-arch-01-start-light.png
   :class: only-light
   :alt: Determined AI system architecture diagram light mode

*Determined AI System Architecture*

Learn more:

-  :ref:`Intro to Determined <introduction-determined>`: Conceptual information about Determined
   including its features and benefits.
-  :ref:`System Architecture <system-architecture>`: Learn about the main components of the
   Determined system architecture.
-  :ref:`Distributed Training <distributed-training-index>`: A conceptual overview of distributed
   training with Determined.

.. toctree::
   :maxdepth: 2
   :glob:

   ./*
