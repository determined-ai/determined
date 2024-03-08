:orphan:

**New Features**

-  Add pre-release NGC-based images to our environments offerings. These images can be pulled from
   `https://hub.docker.com/r/determinedai/pytorch-ngc` or
   `https://hub.docker.com/r/determinedai/tensorflow-ngc`
   Users can build their own images from a given NGC container version by using the
   `build-pytorch-ngc` or `build-tensorflow-ngc` workflows found in our environments `MakeFile`.
   Our environments repo is located at `https://github.com/determined-ai/environments`.

**Improvements**

-  Remove TF 2.8 images from our offerings. Default TF 2.11 images are still available for
   TensorFlow users.
