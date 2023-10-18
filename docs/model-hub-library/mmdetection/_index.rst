.. _model-hub-mmdetection:

#############
 MMDetection
#############

.. _readme: https://github.com/determined-ai/determined/tree/master/model_hub/examples/mmdetection/README.md

`The MMDetection library <https://mmdetection.readthedocs.io/en/latest>`_ is a popular library for
object detection. It provides implementations for many popular object detection approaches like
Faster-RCNN and Mask-RCNN in addition to cutting edge methods from the research community.

**model-hub** makes it easy to use MMDetection with Determined while keeping the developer
experience as close as possible to what it's like working directly with **MMDetection**. Our library
serves as an alternative to the trainer used by MMDetection (see `mmcv's runner
<https://mmcv.readthedocs.io/en/latest/understand_mmcv/runner.html>`_) and provides access to all of
Determined's benefits including:

-  Easy multi-node distributed training with no code modifications. Determined automatically sets up
   the distributed backend for you.
-  Experiment monitoring and tracking, artifact tracking, and :ref:`state-of-the-art hyperparameter
   search <hyperparameter-tuning>` without requiring third-party integrations.
-  :ref:`Automated cluster management, fault tolerance, and job rescheduling <features>` so you
   don't have to worry about provisioning resources or babysitting your experiments.

.. include:: ../../_shared/note-dtrain-learn-more.txt

Given the benefits above, we think this library will be particularly useful to you if any of the
following apply:

-  You want to perform object detection using a powerful integrated platform that will scale easily
   with your needs.
-  You are an Determined user that wants to get started quickly with **MMDetection**.
-  You are a **MMDetection** user that wants to easily run more advanced workflows like multi-node
   distributed training and advanced hyperparameter search.
-  You are a **MMDetection** user looking for a single platform to manage experiments, handle
   checkpoints with automated fault tolerance, and perform hyperparameter search/visualization.
