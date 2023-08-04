:orphan:

**Deprecated Features**

-  API: ``PyTorchTrial`` support for mixed precision using ``NVIDIA/apex`` library is deprecated and
   will be removed in a future version. We recommend users to migrate to Torch Automatic Mixed
   Precision (``torch.cuda.amp``). For more, refer to the
   `examples <https://github.com/determined-ai/determined/tree/0.23.4/harness/tests/experiment/fixtures/pytorch_amp>`_.

-  Images: Environment images will also no longer include ``NVIDIA/apex`` package in a future
   version. User may install this package from the official repository instead.
