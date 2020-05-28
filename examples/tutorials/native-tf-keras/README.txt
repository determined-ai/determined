.. _tutorials_native-api:

Native API Tutorial
===================

The Native API allows developers to seamlessly move between training in a local
development environment and training at cluster-scale on a Determined cluster.
It also provides an interface to train ``tf.keras`` and ``tf.estimator`` models
using idiomatic framework patterns, reducing (or eliminating) the effort to
port model code for use with Determined.

This tutorial describes how a minimal ``tf.keras`` example can be quickly ported
to the Native API, and augmented to launch hyperparameter searches and/or
distributed training jobs.

For a more detailed discussion of how the Native API is implemented,
refer to the :ref:`Native API Topic Guide <model-definitions_native-api>`.

Prerequisites
-------------

- You will need access to a Determined cluster to train your model. If you have
  not yet installed Determined, refer to the :ref:`installation instructions
  <install-cluster>`.
- The ``determined`` Python package should be installed in your local development
  environment, including TensorFlow: ``pip install determined[tensorflow]``.
