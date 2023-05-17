import logging
from typing import Any, Dict

import torch
import torch.nn as nn
from attrdict import AttrDict
from torch.utils.data import Dataset
from torchvision import models

import deepspeed
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

    def __getitem__(self, idx: int) -> torch.Tensor:
        img = self.imgs[idx % self.num_actual_datapoints]
        label = self.labels[idx % self.num_actual_datapoints]
        return img, label


def main(
    core_context: det.core.Context,
    hparams: Dict[str, Any],
) -> None:
    is_chief = core_context.distributed.rank == 0
    hparams = AttrDict(hparams)
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

    fp16 = model_engine.fp16_enabled()
    criterion = nn.CrossEntropyLoss()

    steps_completed = 0
    for op in core_context.searcher.operations():
        while steps_completed < op.length:
            for data in trainloader:
                with dsat.dsat_reporting_context(core_context, op):
                    inputs, labels = data[0].to(model_engine.local_rank), data[1].to(
                        model_engine.local_rank
                    )
                    if fp16:
                        inputs = inputs.half()
                    outputs = model_engine(inputs)
                    loss = criterion(outputs, labels)
                    model_engine.backward(loss)
                    model_engine.step()
                    steps_completed += 1
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
            op.report_completed(loss.item())


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    hparams = info.trial.hparams
    distributed = det.core.DistributedContext.from_deepspeed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams)
