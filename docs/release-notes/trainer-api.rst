:orphan:

**Breaking Changes**

-  ``records_per_epoch`` has been dropped from PyTorch codepaths. We were previously using this
   value internally to estimate epoch lengths. We are now using the chief worker's epoch length as
   the epoch length.

-  ``average_training_metrics`` is no longer configurable. This value previously defaulted to false
   and was dropped to simplify the training API. We always average training metrics now.

**New Features**

-  Introduces :class:`~determined.pytorch.Trainer`, a high-level training API for ``PyTorchTrial``
   that allows for python-side training loop customizations and includes support for off-cluster
   local training.
