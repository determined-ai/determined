:orphan:

**Breaking Changes**

*  Agent: Default visible GPUs now imported from the environment.

   *  The default value for the Determined agent ``--visible-gpus``option is now taken from 
      the environment variables CUDA_VISIBLE_DEVICES or ROCR_VISIBLE_DEVICES, if defined.