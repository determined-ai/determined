import random
from typing import Any, Dict, Tuple

import numpy as np
import torch
import yaml
from torch import nn

from determined import experimental, pytorch


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return torch.Tensor([float(1)])


class NoopPytorchTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext):
        self.context = context

        model = nn.Linear(1, 1, False)
        model.weight.data.fill_(0)

        self.model = context.wrap_model(model)

        opt = torch.optim.SGD(self.model.parameters(), 0.1)
        self.opt = context.wrap_optimizer(opt)

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        w_real = self.model.weight.data[0]

        loss = torch.nn.MSELoss()(self.model(batch), batch)

        self.context.backward(loss)
        self.context.step_optimizer(self.opt)

        print("finished train_batch for rank {}".format(self.context.distributed.get_rank()))

        return {"loss": loss, "w_real": w_real}

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        val = batch[0]
        np_rand = np.random.randint(1, 1000)
        rand_rand = random.randint(0, 1000)
        torch_rand = torch.randint(1000, (1,))
        gpu_rand = torch.randint(1000, (1,), device=self.context.device)

        print("finished evaluate_batch for rank {}".format(self.context.distributed.get_rank()))

        return {
            "validation_error": val,
            "np_rand": np_rand,
            "rand_rand": rand_rand,
            "torch_rand": torch_rand,
            "gpu_rand": gpu_rand,
        }

    def build_training_data_loader(self):
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self):
        return pytorch.DataLoader(OnesDataset(), batch_size=self.context.get_per_slot_batch_size())


if __name__ == "__main__":
    conf = yaml.safe_load(
        """
    description: noop-pytorch-native-api
    data:
      model_type: single_output
    hyperparameters:
      global_batch_size: 32
    scheduling_unit: 1
    searcher:
      name: single
      metric: validation_error
      max_length:
        batches: 3
      smaller_is_better: true
    max_restarts: 0
    min_checkpoint_period:
      batches: 1
    min_validation_period:
      batches: 1
    """
    )
    experimental.create(NoopPytorchTrial, conf, context_dir=".")
