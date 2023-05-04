:orphan:

**Breaking Changes**

-  Experiment: Optimizer has to be an instance of tensorflow.keras.optimizers.legacy.Optimizer starting from Keras 2.11
      -  Experiments now use images with TensorFlow 2.11 by default. TensorFlow users who are not
         explicitly configuring their training image(s) will need to adapt their model code to
         reflect these changes. Users will likely need to use Keras optimizers located in
         ``tensorflow.keras.optimizers.legacy``. Depending on the sophistication of users' model
         code, there may be other breaking changes. Determined is not responsible for these
         breakages. See the `TensorFlow release notes
         <https://github.com/tensorflow/tensorflow/releases/tag/v2.11.0>`_ for more details.

      -  PyTorch users and users who specify custom images should not be affected.
