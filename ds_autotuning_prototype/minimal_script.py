import logging
from typing import Any, Dict, Optional

import deepspeed
import torch
import torch.nn as nn
import torch.nn.functional as F
from attrdict import AttrDict
from dsat import utils
from torch.utils.data import Dataset

import determined as det
from determined.pytorch import DataLoader


class RandDataset(Dataset):
    def __init__(self, num_records: int, dim: int) -> None:
        self.num_records = num_records
        self.records = torch.randn(num_records, dim)

    def __len__(self) -> int:
        return self.num_records

    def __getitem__(self, idx: int) -> torch.Tensor:
        return self.records[idx]


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
    dataset = RandDataset(hparams.num_records, hparams.dim)
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
            for batch in train_loader:
                if fp16:
                    batch = batch.half()
                batch = batch.to(device)
                outputs = utils.dsat_forward(core_context, op, model_engine, batch)
                loss = F.mse_loss(outputs, batch)
                model_engine.backward(loss)
                model_engine.step()
                if model_engine.is_gradient_accumulation_boundary():
                    break

            steps_completed += 1
            if core_context.preempt.should_preempt():
                return
        if is_chief:
            # Just reporting trivial metrics in non DS AT cases for testing purposes.
            metrics_dict = {"non_ds_at_placeholder": 0}
            metrics_dict = utils.dsat_metrics_converter(metrics_dict)
            print("METRICS_DICT", metrics_dict)
            core_context.train.report_training_metrics(
                steps_completed=steps_completed, metrics=metrics_dict
            )
            # TODO: for the Web UI visualization to appear, report_validation_metrics must be
            # called with the relevant metrics passed in. This may require awkward changes in user
            # code or we may need more/extended helper functions.  TBD.
            core_context.train.report_validation_metrics(
                steps_completed=steps_completed, metrics=metrics_dict
            )
            # TODO: Test reporting heterogeneous metrics at different steps
            op.report_completed(metrics_dict)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    latest_checkpoint = info.latest_checkpoint
    hparams = info.trial.hparams
    distributed = det.core.DistributedContext.from_torch_distributed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams)
