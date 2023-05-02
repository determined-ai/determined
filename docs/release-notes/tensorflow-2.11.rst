:orphan:

**Breaking Changes**

-  Images: default images use TensorFlow 2.11.
      -  Experiments now use images with TensorFlow 2.11 by default. TensorFlow users who are not
         explicitly configuring their training image(s) will need to adapt their model code to
         reflect these changes. In some cases, such as with optimizers, there are legacy options.
         The Determined examples have been changed in a few ways: the optimizer kwarg ``lr`` has
         been replaced with ``learning_rate``, and learning rate decay is no longer specified by the
         kwarg ``decay`` but instead with the instantiation of a schedule such as
         ``ExponentialDecay``. Depending on the sophistication of users' model code, there may be
         other breaking changes. Determined is not responsible for these breakages. See the
         `TensorFlow release notes <https://github.com/tensorflow/tensorflow/releases/tag/v2.11.0>`_
         for more details.

      -  PyTorch users and users who specify custom images should not be affected.
