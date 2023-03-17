:orphan:

**Breaking Changes**

-  Trial API: ``on_validation_step_start`` and ``on_validation_step_end`` were previously 
   deprecated callbacks on ``PyTorchTrial`` and ``DeepspeedTrial`` and have been removed. Please 
   use ``on_validation_start`` and ``on_validation_end`` instead. 
