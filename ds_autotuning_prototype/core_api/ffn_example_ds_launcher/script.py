import logging
from typing import Any, Dict

import deepspeed
import determined as det
import torch
import torch.nn as nn
import torch.nn.functional as F
from attrdict import AttrDict
from determined.pytorch import dsat
from torch.utils.data import Dataset


class RandDataset(Dataset):
    def __init__(self, dim: int, num_actual_datapoints: int = 128) -> None:
        self.num_actual_datapoints = num_actual_datapoints
        self.dim = dim
        self.data = torch.randn(self.num_actual_datapoints, self.dim)

    def __len__(self) -> int:
        return 10**6

    def __getitem__(self, idx: int) -> torch.Tensor:
        data = self.data[idx % self.num_actual_datapoints]
        return data


class MinimalModel(nn.Module):
    def __init__(self, dim: int, layers: int) -> None:
        super().__init__()
        self.dim = dim
        layers = [nn.Linear(dim, dim, bias=False) for _ in range(layers)]
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
    deepspeed.init_distributed()
    is_chief = core_context.distributed.rank == 0
    hparams = AttrDict(hparams)
    if is_chief:
        logging.info(f"HPs seen by trial: {hparams}")
    ds_config = dsat.get_ds_config_from_hparams(hparams)
    dataset = RandDataset(hparams.dim)
    model = MinimalModel(hparams.dim, hparams.layers)

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
            for batch in train_loader:
                with dsat.dsat_reporting_context(core_context, op):
                    if fp16:
                        batch = batch.half()
                    batch = batch.to(device)
                    print("batch on device")
                    logging.info(f"ACTUAL BATCH SIZE: {batch.shape[0]}")  # Sanity checking.
                    outputs = model_engine(batch)
                    print("forward complete")
                    loss = F.mse_loss(outputs, batch)
                    model_engine.backward(loss)
                    print("backward complete")
                    model_engine.step()
                    steps_completed += 1
                    print("stepped optimizer")
                if is_chief:
                    metrics_dict = {"loss": loss.item()}
                    core_context.train.report_validation_metrics(
                        steps_completed=steps_completed, metrics=metrics_dict
                    )
                if model_engine.is_gradient_accumulation_boundary():
                    if steps_completed == op.length:
                        break
                if core_context.preempt.should_preempt():
                    return
        if is_chief:
            op.report_completed(metrics_dict)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    hparams = info.trial.hparams
    distributed = det.core.DistributedContext.from_deepspeed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams)
