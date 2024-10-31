:orphan:

**Known Issue**

-  PyTorch has `deprecated
   <https://pytorch.org/tutorials/intermediate/tensorboard_profiler_tutorial.html#use-tensorboard-to-view-results-and-analyze-model-performance>`
   their Profiler TensorBoard Plugin. Our latest environment image comes with PyTorch 2.3, so some
   of the ``tb_plugin`` features may not work. If you are looking to use all the features, we
   suggest using an image with PyTorch that's below version 2.0.
