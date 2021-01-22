from typing import Callable, NewType, Any
import pytorch_lightning as pl


GH = NewType('GH', Callable[[str], Any])


class DETLightningModule(pl.LightningModule):
    def __init__(self, get_hparam: GH, *args, **kwargs):  # Py QUESTION should I add this is kwarg?
        super().__init__(*args, **kwargs)
        self.get_hparam = get_hparam


class PTLAdapter(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext, lm: DETLightningModule) -> None:
        super().__init__(context)
        self.lm = lm(context.get_hparam)  # TODO pass in context.get_hparam and dataloaders?
        self.context = context
        self.model = self.context.wrap_model(self.lm)
        self.optimizer = self.context.wrap_optimizer(self.lm.configure_optimizers())

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        rv = self.lm.training_step(batch, batch_idx)

        # TODO option to set loss
        self.context.backward(rv['loss'])
        self.context.step_optimizer(self.optimizer)
        return rv

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        return self.lm.validation_step(batch)
