Native API Tutorial (Experimental)
==================================

.. warning::

    This API is currently experimental and subject to change.

The Native API allows developers to seamlessly move between from training in
local Python scripts to training at cluster-scale on a Determined cluster. It
also provides an interface to train ``tf.keras`` and ``tf.estimator`` models
using idiomatic framework patterns, reducing (or eliminating) the effort to
port model code for use with Determined.

This tutorial describes how a minimal ``tf.keras`` example can be quickly
ported to the Native API to train on a Determined cluster, and augmented to
launch hyperparameter searches and/or distributed training jobs.

.. TODO: Add a link to Native topic guide.

Prerequisites
-------------

- You will need access to a Determined cluster to train your model. If you have
  not yet installed Determined, refer to the :ref:`installation instructions
  <install-cluster>`.
- The ``determined`` Python package installed in your local development
  environment, including TensorFlow: ``pip install determined[tensorflow]``.
