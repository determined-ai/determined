import pytest
import torch

from determined import errors, pytorch
from determined.common import check
from tests.experiment.fixtures import pytorch_onevar_model


class TestPyTorchContext:
    def setup_method(self) -> None:
        self.config = {"hyperparameters": {"global_batch_size": 4, "dataloader_type": "determined"}}
        context = pytorch.PyTorchTrialContext.from_config(self.config)
        assert isinstance(context, pytorch.PyTorchTrialContext)
        self.context = context

    def test_from_config(self) -> None:
        trial = pytorch_onevar_model.OneVarTrial(self.context)

        train_ds = trial.build_training_data_loader()
        for epoch_idx in range(3):
            for batch_idx, batch in enumerate(train_ds):
                metrics = trial.train_batch(batch, epoch_idx, batch_idx)
                # Verify the training is correct.
                pytorch_onevar_model.OneVarTrial.check_batch_metrics(
                    metrics,
                    batch_idx,
                    metric_keyname_pairs=(("loss", "loss_exp"), ("w_after", "w_exp")),
                )

        eval_ds = trial.build_validation_data_loader()
        for batch in eval_ds:
            metrics = trial.evaluate_batch(batch)

    def test_average_gradients(self) -> None:
        with pytest.raises(check.CheckFailedError):
            self.context._average_gradients(None, 0)
        assert self.context._average_gradients(None, 1) is None

    def test_training_not_started(self) -> None:
        with pytest.raises(errors.InternalException):
            self.context.is_epoch_start()
        with pytest.raises(errors.InternalException):
            self.context.is_epoch_end()
        with pytest.raises(errors.InternalException):
            self.context.current_train_batch()
        with pytest.raises(errors.InternalException):
            self.context.current_train_epoch()
        self.context.env.managed_training = True
        with pytest.raises(errors.InternalException):
            self.context._should_communicate_and_update()

    def test_wrap_scaler(self) -> None:
        if torch.cuda.is_available():
            scaler = torch.cuda.amp.GradScaler()  # type: ignore # GradScaler.__init__ is untyped
            assert scaler == self.context.wrap_scaler(scaler)
            assert scaler == self.context._scaler
        else:
            with pytest.raises(check.CheckFailedError):
                self.context.wrap_scaler(None)
