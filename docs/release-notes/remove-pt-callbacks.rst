:orphan:

**Breaking Changes**

-  Trial API: ``on_validation_step_start`` and ``on_validation_step_end`` callbacks on
   ``PyTorchTrial`` and ``DeepspeedTrial`` were deprecated in 0.12.12 (Jul 2020) and have been
   removed. Please use ``on_validation_start`` and ``on_validation_end`` instead.
