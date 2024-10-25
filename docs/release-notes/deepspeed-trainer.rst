:orphan:

**New Features**

-  DeepSpeed Trainer: Add Trainer API to DeepSpeedTrial. Users can now iterate on DeepSpeed models
   off-cluster. An example of DeepSpeed Trainer API has been added in the `examples/deepspeed/dcgan`
   directory.

**Breaking Change**

-  DeepSpeedContext `env` removal: DeepSpeedContext no longer keeps track of `det.EnvContext`. Users
   who need direct access to variables tracked in the `det.EnvContext` should refer to the helper
   functions in `DeepSpeedContext`.
