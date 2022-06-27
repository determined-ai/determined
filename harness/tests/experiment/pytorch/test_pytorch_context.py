import unittest

import torch

from determined import pytorch
from determined.common.check import CheckFailedError
from determined.errors import InternalException
from tests.experiment.fixtures import pytorch_onevar_model


class TestPyTorchContext(unittest.TestCase):
    def setUp(self) -> None:
        self.config = {"hyperparameters": {"global_batch_size": 4, "dataloader_type": "determined"}}
        self.context: pytorch.PyTorchTrialContext = pytorch.PyTorchTrialContext.from_config(
            self.config
        )
        assert isinstance(self.context, pytorch.PyTorchTrialContext)

        trial = pytorch_onevar_model.OneVarTrial(self.context)

        train_ds = trial.build_training_data_loader()
        for epoch_idx in range(3):
            for batch_idx, batch in enumerate(train_ds):
                metrics = trial.train_batch(batch, epoch_idx, batch_idx)
                # Verify the training is correct.
                pytorch_onevar_model.OneVarTrial.check_batch_metrics(metrics, batch_idx)

        eval_ds = trial.build_validation_data_loader()
        for batch in eval_ds:
            metrics = trial.evaluate_batch(batch)

    def test_average_gradients(self) -> None:
        self.assertRaises(CheckFailedError, self.context._average_gradients, None, 0)
        self.assertIsNone(self.context._average_gradients(None, 1))

    def test_training_not_started(self) -> None:
        self.assertRaises(InternalException, self.context.is_epoch_start)
        self.assertRaises(InternalException, self.context.is_epoch_end)
        self.assertRaises(InternalException, self.context.current_train_batch)
        self.assertRaises(InternalException, self.context.current_train_epoch)
        self.context.env.managed_training = True
        self.assertRaises(InternalException, self.context._should_communicate_and_update)

    def test_wrap_scalar(self) -> None:
        scaler = 1
        if not torch.cuda.is_available():
            self.assertRaises(CheckFailedError, self.context.wrap_scaler, scaler)
        else:
            self.assertEqual(scaler, self.context.wrap_scaler(scaler))
            self.assertEqual(scaler, self.context._scaler)
