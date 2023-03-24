import math
from collections import defaultdict

import torch.nn as nn
from attrdict import AttrDict
from byol_pytorch import BYOL
from lars import LARS
from torch.optim import SGD, Optimizer


# LARS code adapted from https://github.com/untitled-ai/self_supervised/blob/master/moco.py
# License is replicated below:
# ===============================================================================
# MIT License
# Copyright (c) 2020 Untiled AI
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
# ===============================================================================
def build_byol_optimizer(hparams: AttrDict, model: nn.Module) -> Optimizer:
    """
    Build optimizer for BYOL self-supervised network, including backbone.
    """
    regular_parameters = []
    excluded_parameters = []
    for name, parameter in model.named_parameters():
        if parameter.requires_grad is False:
            continue
        if any(x in name for x in [".bn", ".bias"]):
            excluded_parameters.append(parameter)
        else:
            regular_parameters.append(parameter)
    param_groups = [
        {"params": regular_parameters, "use_lars": True},
        {
            "params": excluded_parameters,
            "use_lars": False,
            "weight_decay": 0,
        },
    ]
    return LARS(
        param_groups,
        lr=hparams.self_supervised.learning_rate.base,
        eta=hparams.self_supervised.lars_eta,
        momentum=hparams.self_supervised.momentum,
        weight_decay=hparams.self_supervised.weight_decay,
    )


def build_cls_optimizer(hparams: AttrDict, lr: float, model: nn.Module) -> Optimizer:
    """
    Build optimizer for classifier head used for evaluation.

    In BYOL paper, multiple LRs are evaluated and the best is taken.  Thus,
    this build function is parameterized by LR.
    """
    return SGD(model.parameters(), lr, momentum=hparams.classifier.momentum, nesterov=True)


def reset_model_parameters(model: nn.Module) -> None:
    for layer in model.children():
        if hasattr(layer, "reset_parameters"):
            layer.reset_parameters()  # type: ignore


def reset_sgd_optimizer(opt: Optimizer) -> None:
    """
    Reset SGD optimizer momentum buffer.
    """
    opt.state = defaultdict(dict)


def set_learning_rate_warmup_cosine_anneal(
    hparams: AttrDict,
    optimizer: Optimizer,
    global_batch_size: int,
    batch_idx: int,
    batches_per_epoch: int,
) -> None:
    """
    Cosine annealing with warmup, as described in BYOL paper.
    """
    # Learning rate scales linearly with batch size, is equal to base when global_batch_size==base_batch_size.
    p = hparams.self_supervised.learning_rate
    base_lr = p.base * (global_batch_size / p.base_batch_size)
    fractional_epoch = batch_idx / batches_per_epoch
    if fractional_epoch <= p.warmup_epochs:
        # Warmup domain.
        adjusted_lr = base_lr * (fractional_epoch / p.warmup_epochs)
    else:
        # Cosine annealing domain.
        cosine_progress = (fractional_epoch - p.warmup_epochs) / (
            hparams.total_epochs - p.warmup_epochs
        )
        cosine_multiplier = 1 + math.cos(math.pi * cosine_progress)
        adjusted_lr = base_lr * cosine_multiplier
    for param_group in optimizer.param_groups:
        param_group["lr"] = adjusted_lr


def set_ema_beta_cosine_anneal(
    hparams: AttrDict, byol_model: BYOL, batch_idx: int, batches_per_epoch: int
) -> None:
    """
    Anneals the exponential moving average coefficient, as described in the BYOL paper.
    """
    fractional_epoch = batch_idx / batches_per_epoch
    # 1 − (1 − τbase) · (cos(πk/K) + 1)/2
    progress = fractional_epoch / hparams.total_epochs
    ema_beta = 1 - (
        (1 - hparams.self_supervised.moving_average_decay_base)
        * (math.cos(math.pi * progress) + 1)
        / 2
    )
    byol_model.target_ema_updater.beta = ema_beta
