.. _model-hub-mmdetection-api:

#################
 MMDetection API
#################

***************************
 ``model_hub.mmdetection``
***************************

.. _mmdettrial:

.. _readme: https://github.com/determined-ai/determined/tree/master/model_hub/examples/mmdetection/README.md

.. autoclass:: model_hub.mmdetection.MMDetTrial

Simlar to using the MMDetection library directly, the main way users customize an experiment is by
modifying the MMDetection config. We detail how to configure MMDetection through the Determined
experiment configuration in the readme_.

Helper Functions
================

.. automodule:: model_hub.mmdetection
   :members: get_pretrained_ckpt_path, GCSBackend, S3Backend
