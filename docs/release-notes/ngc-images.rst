:orphan:

**New Features**

-  Add Add early access NVIDIA NGC-based images to our environments offerings. These images can be
   pulled from `https://hub.docker.com/r/determinedai/pytorch-ngc` or
   `https://hub.docker.com/r/determinedai/tensorflow-ngc`. By downloading and using these images,
   you accept the terms and conditions of all third-party software licenses contained within,
   including the `NVIDIA Deep Learning Container
   License.`<https://developer.download.nvidia.com/licenses/NVIDIA_Deep_Learning_Container_License.pdf>`_
   Users can build their own images from a given NGC container version by using the
   `build-pytorch-ngc` or `build-tensorflow-ngc` workflows found in our environments `MakeFile`. Our
   environments repo is located at `https://github.com/determined-ai/environments`.

**Improvements**

-  Remove TF 2.8 images from our offerings. Default TF 2.11 images are still available for
   TensorFlow users.
