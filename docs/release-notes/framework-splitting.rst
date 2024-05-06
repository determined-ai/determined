:orphan:

**Breaking Change**

-  Default images are being changed to support only PyTorch by default. TensorFlow users must
   configure their experiments to target our non-default TensorFlow images. Details on this process
   can be found at :ref:`set-environment-images`

-  Our new default images are built off of Nvidia NGC images. We select an NGC version, however
   users can build their own images from an NGC version of their choice. See :ref:`ngc-version`
