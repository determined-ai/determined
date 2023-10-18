from typing import Any, Dict

import torch
from torch import nn

from determined.pytorch import TorchData


class Flatten(nn.Module):
    def forward(self, *args: TorchData, **kwargs: Any) -> torch.Tensor:
        assert len(args) == 1
        x = args[0]
        assert isinstance(x, torch.Tensor)
        return x.contiguous().view(x.size(0), -1)


def build_model(hparams: Dict) -> nn.Module:
    return nn.Sequential(
        nn.Conv2d(1, hparams["n_filters1"], 3, 1),
        nn.ReLU(),
        nn.Conv2d(
            hparams["n_filters1"],
            hparams["n_filters2"],
            3,
        ),
        nn.ReLU(),
        nn.MaxPool2d(2),
        nn.Dropout2d(hparams["dropout1"]),
        Flatten(),
        nn.Linear(144 * hparams["n_filters2"], 128),
        nn.ReLU(),
        nn.Dropout2d(hparams["dropout2"]),
        nn.Linear(128, 10),
        nn.LogSoftmax(),
    )
