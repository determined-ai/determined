import random
from typing import Any, Dict, Sequence, Tuple, Union

import numpy as np
import torch
from torch import nn

from determined.pytorch import DataLoader, PyTorchTrial

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]

LR_START = 0.1
INPUT = 1


class IndexDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return torch.Tensor([float(INPUT)])


def yellow(msg):
    return "\x1b[33m" + msg + "\x1b[m"


class NoopPytorchTrial(PyTorchTrial):
    def __init__(self, context):
        self.context = context

        model = nn.Linear(1, 1, False)
        # initialize weights to 0
        model.weight.data.fill_(0)
        print("weight starts at:", model.weight.data[0])

        self.model = context.wrap_model(model)

        opt = torch.optim.SGD(self.model.parameters(), LR_START)
        self.opt = context.wrap_optimizer(opt)

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        # Figure what the weight should be right now
        w_real = self.model.weight.data[0]

        loss = torch.nn.L1Loss()(self.model(batch), batch)

        self.context.backward(loss)
        self.context.step_optimizer(self.opt)

        print(yellow("numpy"), np.random.randint(1000))
        print(yellow("random"), random.randint(0, 1000))
        print(yellow("torch"), torch.randint(1000, (1,)))

        # check what the weight actually is
        return {"loss": loss, "w_real": w_real}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        # Return something... anything.
        val = batch[0]
        rand_int = np.random.randint(1, 100)
        return {"validation_error": val, "rand_int": rand_int}

    def build_training_data_loader(self):
        return DataLoader(IndexDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self):
        return DataLoader(IndexDataset(), batch_size=self.context.get_per_slot_batch_size())

    # def build_callbacks(self):
    #     return {
    #         "rngsaver": rngsaver.RNGSaver(self.context.distributed.get_local_rank())
    #     }


if __name__ == "__main__":
    import yaml

    config = yaml.safe_load(
        """
    description: rb-onevar-pytorch
    data:
      model_type: single_output
    entrypoint: model_def:NoopPytorchTrial
    hyperparameters:
      global_batch_size: 32
    batches_per_step: 1
    searcher:
      name: single
      metric: validation_error
      max_steps: 100
      smaller_is_better: true
    min_checkpoint_period: 1
    min_validation_period: 1
    max_restarts: 0
    """
    )

    import determined.experimental

    determined.experimental.create(
        trial_def=NoopPytorchTrial,
        config=config,
        local=False,
        test=False,
        context_dir=".",
        master_url="localhost:8080",
    )
