:orphan:

**New Feature**

-  PyTorchTrial and DeepSpeedTrial APIs: Deprecated ``TorchWriter`` and added a PyTorch
   ``SummaryWriter`` object to ``PyTorchTrialContext`` and ``DeepSpeedTrialContext`` that we manage
   on behalf of users. See :func:`~determined.pytorch.PyTorchTrialContext.get_tensorboard_writer`
   for details.
