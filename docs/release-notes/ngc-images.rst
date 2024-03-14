:orphan:

**New Features**

-  Include early-access NVIDIA NGC-based images in our environment offerings. These images are
   accessible from `pytorch-ngc <https://hub.docker.com/r/determinedai/pytorch-ngc>`_ or
   `tensorflow-ngc <https://hub.docker.com/r/determinedai/tensorflow-ngc>`_. By downloading and
   using these images, users acknowledge and agree to the terms and conditions of all third-party
   software licenses contained within, including the `NVIDIA Deep Learning Container License
   <https://developer.download.nvidia.com/licenses/NVIDIA_Deep_Learning_Container_License.pdf>`__.
   Users can build their own images from a specified NGC container version by using the
   ``build-pytorch-ngc`` or ``build-tensorflow-ngc`` workflows located in our environments
   ``MakeFile`` in the `environments repository <https://github.com/determined-ai/environments>`_.

**Improvements**

-  Eliminate TensorFlow 2.8 images from our offerings. Default TensorFlow 2.11 images remain
   available for TensorFlow users.
