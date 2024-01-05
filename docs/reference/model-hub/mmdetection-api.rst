.. _model-hub-mmdetection-api:

#################
 MMDetection API
#################

***************************
 ``model_hub.mmdetection``
***************************

.. _mmdettrial:

.. _readme: https://github.com/determined-ai/determined-examples/blob/main/model_hub/mmdetection/README.md

.. autoclass:: model_hub.mmdetection.MMDetTrial

Simlar to using the MMDetection library directly, the main way users customize an experiment is by
modifying the MMDetection config. To find out how to configure MMDetection using the
:ref:`experiment configuration <experiment-configuration>` file, visit the readme_.

Helper Functions
================

.. automodule:: model_hub.mmdetection
   :members: get_pretrained_ckpt_path, GCSBackend, S3Backend
