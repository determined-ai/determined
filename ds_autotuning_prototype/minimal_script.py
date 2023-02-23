import logging
import sys
from typing import Any, Dict, Optional

import deepspeed
import determined as det
import torch
import torch.nn as nn
import torch.nn.functional as F
from attrdict import AttrDict
from dsat import utils
from torch.utils.data import Dataset


class RandDataset(Dataset):
    def __init__(self, dim: int) -> None:
        self.dim = dim

    def __len__(self) -> int:
        return 2 ** 16 - 1

    def __getitem__(self, idx: int) -> torch.Tensor:
        return torch.randn(self.dim)


class MinimalModel(nn.Module):
    def __init__(self, dim: int, layers: int) -> None:
        super().__init__()
        self.dim = dim
        layers = [nn.Linear(dim, dim) for _ in range(layers)]
        self.model = nn.ModuleList(layers)

    def forward(self, inputs: torch.Tensor) -> torch.Tensor:
        outputs = inputs
        for layer in self.model:
            outputs = layer(outputs)
        return outputs


def main(
    core_context: det.core.Context,
    hparams: Dict[str, Any],
) -> None:
    is_chief = core_context.distributed.rank == 0
    hparams = AttrDict(hparams)
    if is_chief:
        logging.info(f"HPs seen by trial: {hparams}")
    # Hack for clashing 'type' key. Need to change config parsing behavior so that
    # user scripts don't need to inject helper functions like this.
    ds_config = utils.lower_case_dict_key(hparams.ds_config, "TYPE")
    dataset = RandDataset(hparams.dim)
    model = MinimalModel(hparams.dim, hparams.layers)

    deepspeed.init_distributed()
    model_engine, optimizer, train_loader, __ = deepspeed.initialize(
        model=model,
        model_parameters=model.parameters(),
        training_data=dataset,
        config=ds_config,
    )
    fp16 = model_engine.fp16_enabled()
    # DeepSpeed uses the local_rank as the device, for some reason.
    device = model_engine.device

    steps_completed = 0
    for op in core_context.searcher.operations():
        while steps_completed < op.length:
            steps_completed += 1
            # A potential gotcha: steps_completed must not be altered within the below context.
            # Probably obvious from the usage, but should be noted in docs.
            with utils.dsat_reporting_context(core_context, op, steps_completed):
                for batch in train_loader:
                    if fp16:
                        batch = batch.half()
                    batch = batch.to(device)
                    logging.info(f"BATCH SIZE: {batch.shape[0]}")  # Sanity checking.
                    # outputs = utils.dsat_forward(
                    #     core_context, op, model_engine, steps_completed, batch
                    # )
                    outputs = model_engine(batch)
                    loss = F.mse_loss(outputs, batch)
                    model_engine.backward(loss)
                    model_engine.step()
                    if model_engine.is_gradient_accumulation_boundary():
                        break

            if is_chief:
                metrics_dict = {"loss": loss.item()}
                metrics_dict = utils.dsat_metrics_converter(metrics_dict)
                core_context.train.report_validation_metrics(
                    steps_completed=steps_completed, metrics=metrics_dict
                )
                # TODO: Test reporting heterogeneous metrics at different steps
            if core_context.preempt.should_preempt():
                return
        if is_chief:
            op.report_completed(metrics_dict)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    hparams = info.trial.hparams
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams)
