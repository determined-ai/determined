:orphan:

**Deprecated Features**

-  API: Support for mixed precision in ``PyTorchTrial`` using the ``NVIDIA/apex`` library is
   deprecated and will be removed in a future version of Determined. Users should transition to
   Torch Automatic Mixed Precision (``torch.cuda.amp``). For examples, refer to the `examples
   <https://github.com/determined-ai/determined/tree/0.23.4/harness/tests/experiment/fixtures/pytorch_amp>`_.

-  Images: Likewise, environment images will no longer include the ``NVIDIA/apex`` package in a
   future version of Determined. If needed, users can install it from the official repository.
