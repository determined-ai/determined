:orphan:

**Known Issue**

-  PyTorch has `deprecated
   <https://pytorch.org/tutorials/intermediate/tensorboard_profiler_tutorial.html#use-tensorboard-to-view-results-and-analyze-model-performance>`
   their Profiler TensorBoard Plugin (``tb_plugin``), so some features may not be compatible with
   PyTorch 2.0 and above. Our current default environment image comes with PyTorch 2.3. If users are
   experiencing issues with this plugin, we suggest using an image with a PyTorch version earlier
   than 2.0.
