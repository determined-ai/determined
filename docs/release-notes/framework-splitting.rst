:orphan:

**Breaking Change**

-  Images: The default environment includes images that support PyTorch. TensorFlow users must
   configure their experiments to target our non-default TensorFlow images. Details on this process
   can be found at :ref:`set-environment-images`

-  Images: Our new default images are based on Nvidia NGC. While we provide a recommended NGC
   version, users have the flexibility to build their own images using any NGC version that meets
   their specific requirements. For more information, visit :ref:`ngc-version`
