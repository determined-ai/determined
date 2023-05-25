import logging
import uuid
from typing import Any, Optional, Tuple

import attrdict
import deepspeed
import numpy as np
import torch
import torch.nn as nn
from torch.utils.data import Dataset
from torchvision import models

import determined as det
from determined.pytorch import dsat


class RandImageNetDataset(Dataset):
    """
    A fake, ImageNet-like dataset which only actually contains `num_actual_datapoints` independent
    datapoints, but pretends to have the number reported in `__len__`. Used for speed and
    simplicity. Replace with your own ImageNet-like dataset as desired.
    """

    def __init__(self, num_actual_datapoints: int = 128) -> None:
        self.num_actual_datapoints = num_actual_datapoints
        self.imgs = torch.randn(self.num_actual_datapoints, 3, 224, 224)
        self.labels = torch.randint(1000, size=(self.num_actual_datapoints,))

    def __len__(self) -> int:
        return 10**6

    def __getitem__(self, idx: int) -> Tuple[torch.Tensor, torch.Tensor]:
        img = self.imgs[idx % self.num_actual_datapoints]
        label = self.labels[idx % self.num_actual_datapoints]
        return img, label


def main(
    core_context: det.core.Context,
    hparams: attrdict.AttrDict,
    latest_checkpoint: Optional[uuid.UUID],
) -> None:
    is_chief = core_context.distributed.rank == 0
    ds_config = dsat.get_ds_config_from_hparams(hparams)
    deepspeed.init_distributed()

    trainset = RandImageNetDataset()
    model = getattr(models, hparams.model_name)()
    parameters = filter(lambda p: p.requires_grad, model.parameters())

    model_engine, _, trainloader, _ = deepspeed.initialize(
        model=model,
        model_parameters=parameters,
        training_data=trainset,
        config=ds_config,
    )
    # Restore from latest checkpoint, if any.
    if latest_checkpoint is not None:
        with core_context.checkpoint.restore_path(storage_id=latest_checkpoint) as path:
            model_engine.load_checkpoint(path)

    fp16 = model_engine.fp16_enabled()
    criterion = nn.CrossEntropyLoss()

    steps_completed = 0
    local_loss_bucket = []
    for op in core_context.searcher.operations():
        while steps_completed < op.length:
            for data in trainloader:
                with dsat.dsat_reporting_context(core_context, op):
                    inputs, labels = data
                    inputs, labels = inputs.to(model_engine.local_rank), labels.to(
                        model_engine.local_rank
                    )
                    if fp16:
                        inputs = inputs.half()
                    outputs = model_engine(inputs)
                    loss = criterion(outputs, labels)
                    local_loss_bucket.append(loss.item())
                    model_engine.backward(loss)
                    model_engine.step()

                # Only increment `steps_completed` when an actual optimizer step is taken,
                # accounting for the gradient accumulation rate.
                if model_engine.is_gradient_accumulation_boundary():
                    steps_completed += 1
                    # Metrics reporting.
                    if not steps_completed % hparams.metric_reporting_rate:
                        mean_local_loss = np.array(local_loss_bucket).mean()
                        local_loss_bucket = []
                        gathered_losses = core_context.distributed.gather(mean_local_loss)
                        if is_chief:
                            mean_global_loss = np.array(gathered_losses).mean()
                            metrics_dict = {"loss": mean_global_loss}
                            core_context.train.report_training_metrics(
                                steps_completed=steps_completed, metrics=metrics_dict
                            )
                    # Checkpointing.
                    if not steps_completed % hparams.checkpoint_rate:
                        metadata = {"steps_completed": steps_completed}
                        with core_context.checkpoint.store_path(metadata=metadata, shard=True) as (
                            path,
                            _,
                        ):
                            model_engine.save_checkpoint(path)
                        # Preemption after checkpointing.
                        if core_context.preempt.should_preempt():
                            return
                    # Completion.
                    if steps_completed == op.length:
                        if is_chief:
                            op.report_completed(mean_global_loss)
                        return


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    latest_checkpoint = info.latest_checkpoint
    hparams = attrdict.AttrDict(info.trial.hparams)
    distributed = det.core.DistributedContext.from_deepspeed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams, latest_checkpoint)
