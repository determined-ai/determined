"""
This example is based on Determined's MNIST PyTorch example. 

This file is a how-to example for multiple learning rate schedulers
in Determined. 

"""

from typing import Any, Dict, Sequence, Tuple, Union, cast

import torch
from torch import nn
from torch.optim.lr_scheduler import _LRScheduler

from layers import Flatten  # noqa: I100

import determined as det
from determined.pytorch import DataLoader, PyTorchTrial, reset_parameters, LRScheduler
import data
import torchvision

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


class MultiLRScheduler(_LRScheduler):
    def __init__(self, lr1, lr2, optimizer, last_epoch=-1):
        self.lr1 = lr1
        self.lr2 = lr2
        super(MultiLRScheduler, self).__init__(optimizer, last_epoch)

    def load_state_dict(self, state_dict):
        lr1_state = state_dict["lr1"]
        lr2_state = state_dict["lr2"]
        del state_dict["lr1"]
        del state_dict["lr2"]

        super().load_state_dict(state_dict)
        self.lr1.load_state_dict(lr1_state)
        self.lr2.load_state_dict(lr2_state)


    def state_dict(self):
        state = super().state_dict()
        state["lr1"] = self.lr1.state_dict()
        state["lr2"] = self.lr2.state_dict()
        state['last_epoch'] = self.last_epoch
        state['_step_count'] = self._step_count

        return state
    
    def step(self, epoch=None):
        if epoch is None and self.last_epoch < 0:
            '''
            During initalization, PyTorch schedulers call .step().
            Therefore, self.lr1 and self.lr2 have already had the initial .step()
            called. We need to then just set the main class variables.
            '''
            self._step_count = 1
            self.last_epoch = 0
            
        else:
            if epoch is None:
                epoch = self.last_epoch + 1
            self.last_epoch = epoch
        
            self.lr1.step()
            self.lr2.step()
        
        self._last_lr = [group['lr'] for group in self.optimizer.param_groups]


class MNistTrial(PyTorchTrial):
    def __init__(self, context: det.TrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

    def build_training_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        dataset = data.get_dataset(self.download_directory, train=True)
        return DataLoader(dataset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            self.download_directory = data.download_dataset(
                download_directory=self.download_directory,
                data_config=self.context.get_data_config(),
            )
            self.data_downloaded = True

        validation_data = data.get_dataset(self.download_directory, train=False)
        return DataLoader(validation_data, batch_size=self.context.get_per_slot_batch_size())

    def create_lr_scheduler(self, optimizer: torch.optim.Optimizer):
        self.Lr2 = torch.optim.lr_scheduler.CosineAnnealingLR(optimizer, 100)
        self.Lr1 = torch.optim.lr_scheduler.StepLR(optimizer, 1)
        
        self.combined_lrs = MultiLRScheduler(self.Lr1, self.Lr2, optimizer)
        # Because we are calling .step() ourselves in our MultiLRScheduler we need to 
        # set the StepMode to MANUAL_STEP.
        return LRScheduler(self.combined_lrs, step_mode=LRScheduler.StepMode.MANUAL_STEP)

    def build_model(self) -> nn.Module:
        model = nn.Sequential(
            nn.Conv2d(1, self.context.get_hparam("n_filters1"), 3, 1),
            nn.ReLU(),
            nn.Conv2d(
                self.context.get_hparam("n_filters1"), self.context.get_hparam("n_filters2"), 3,
            ),
            nn.ReLU(),
            nn.MaxPool2d(2),
            nn.Dropout2d(self.context.get_hparam("dropout1")),
            Flatten(),
            nn.Linear(144 * self.context.get_hparam("n_filters2"), 128),
            nn.ReLU(),
            nn.Dropout2d(self.context.get_hparam("dropout2")),
            nn.Linear(128, 10),
            nn.LogSoftmax(),
        )

        # If loading backbone weights, do not call reset_parameters() or
        # call before loading the backbone weights.
        reset_parameters(model)
        return model

    def optimizer(self, model: nn.Module) -> torch.optim.Optimizer:  # type: ignore
        self.optimizer = torch.optim.Adadelta(model.parameters(), lr=self.context.get_hparam("learning_rate"))
        
        return self.optimizer

    def train_batch(
        self, batch: TorchData, model: nn.Module, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = model(data)
        loss = torch.nn.functional.nll_loss(output, labels)

        self.combined_lrs.step()

        return {"loss": loss}

    def evaluate_batch(self, batch: TorchData, model: nn.Module) -> Dict[str, Any]:
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch

        output = model(data)
        validation_loss = torch.nn.functional.nll_loss(output, labels).item()

        pred = output.argmax(dim=1, keepdim=True)
        accuracy = pred.eq(labels.view_as(pred)).sum().item() / len(data)

        return {"validation_loss": validation_loss, "accuracy": accuracy}
