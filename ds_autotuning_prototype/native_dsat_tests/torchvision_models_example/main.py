import argparse
import logging
import pathlib
import shutil
from typing import Any, Dict, Optional

import deepspeed
import determined as det
import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
import utils
from attrdict import AttrDict
from torch.utils.data import Dataset
from torchvision import models


class RandImageNetDataset(Dataset):
    def __init__(self, num_actual_datapoints: int = 128) -> None:
        self.num_actual_datapoints = num_actual_datapoints
        self.imgs = torch.randn(self.num_actual_datapoints, 3, 224, 224)
        self.labels = torch.randint(1000, size=(self.num_actual_datapoints,))

    def __len__(self) -> int:
        return 2 ** 16

    def __getitem__(self, idx: int) -> torch.Tensor:
        img = self.imgs[idx % self.num_actual_datapoints]
        label = self.labels[idx % self.num_actual_datapoints]
        return img, label


def parse_args():
    parser = argparse.ArgumentParser()
    # Include DeepSpeed configuration arguments
    parser = deepspeed.add_config_arguments(parser)
    # Need to absorb (and do nothing with) a local_rank arg when running autotuning.
    parser.add_argument("--local_rank", type=int, default=None)

    args = parser.parse_args()

    return args


def report_and_save_native_autotuning_results(
    core_context: det.core.Context, path: pathlib.Path = pathlib.Path(".")
) -> None:
    results = utils.DSAutotuningResults(path=path)
    ranked_results_dicts = results.get_ranked_results_dicts()
    for rank, results_dict in enumerate(ranked_results_dicts):
        metrics = results_dict["metrics"]
        ds_config = results_dict["exp_config"]["ds_config"]
        reported_metrics = utils.get_flattened_dict({**metrics, **ds_config})
        core_context.train.report_validation_metrics(
            steps_completed=rank,
            metrics=reported_metrics,
        )

    checkpoint_metadata_dict = {"steps_completed": len(ranked_results_dicts) - 1}
    with core_context.checkpoint.store_path(checkpoint_metadata_dict) as (
        ckpt_path,
        storage_id,
    ):
        for autotuning_dir in ("autotuning_exps", "autotuning_results"):
            src_path = pathlib.Path(autotuning_dir)
            shutil.copytree(
                src=src_path,
                dst=pathlib.Path(ckpt_path).joinpath(autotuning_dir),
            )


def main(
    core_context: det.core.Context,
    hparams: Dict[str, Any],
) -> None:
    # Native DS AT passes args and requires annoyingly specific behavior.
    args = parse_args()
    is_chief = core_context.distributed.rank == 0
    hparams = AttrDict(hparams)
    # TODO: Remove hack for seeing actual used HPs after Web UI is fixed.
    if is_chief:
        logging.info(f"HPs seen by trial: {hparams}")
    # Hack for clashing 'type' key. Need to change config parsing behavior so that
    # user scripts don't need to inject helper functions like this.

    deepspeed.init_distributed()

    ########################################################################
    # The output of torchvision datasets are PILImage images of range [0, 1].
    # We transform them to Tensors of normalized range [-1, 1].
    # .. note::
    #     If running on Windows and you get a BrokenPipeError, try setting
    #     the num_worker of torch.utils.data.DataLoader() to 0.

    trainset = RandImageNetDataset()

    trainloader = torch.utils.data.DataLoader(trainset, batch_size=16, shuffle=True, num_workers=2)

    model_dict = {
        "resnet152": models.resnet152,
        "wide_resnet101_2": models.wide_resnet101_2,
        "vgg19": models.vgg19,
        "regnet_x_32gf": models.regnet_x_32gf,
    }

    net = model_dict[hparams.model_name]()

    parameters = filter(lambda p: p.requires_grad, net.parameters())

    # Initialize DeepSpeed to use the following features
    # 1) Distributed model
    # 2) Distributed data loader
    # 3) DeepSpeed optimizer
    model_engine, optimizer, trainloader, __ = deepspeed.initialize(
        model=net,
        model_parameters=parameters,
        training_data=trainset,
        args=args,
    )

    fp16 = model_engine.fp16_enabled()
    print(f"fp16={fp16}")

    ########################################################################
    # 3. Define a Loss function and optimizer
    # ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
    # Let's use a Classification Cross-Entropy loss and SGD with momentum.

    criterion = nn.CrossEntropyLoss()

    ########################################################################
    # 4. Train the network
    # ^^^^^^^^^^^^^^^^^^^^
    #
    # This is when things start to get interesting.
    # We simply have to loop over our data iterator, and feed the inputs to the
    # network and optimize.

    device = model_engine.device

    steps_completed = 0
    for op in core_context.searcher.operations():
        while steps_completed < op.length:
            steps_completed += 1
            for data in trainloader:
                # get the inputs; data is a list of [inputs, labels]
                inputs, labels = data[0].to(model_engine.local_rank), data[1].to(
                    model_engine.local_rank
                )
                logging.info(f"ACTUAL BATCH SIZE: {inputs.shape[0]}")  # Sanity checking.
                if fp16:
                    inputs = inputs.half()

                outputs = model_engine(inputs)
                loss = criterion(outputs, labels)

                model_engine.backward(loss)
                model_engine.step()
    report_and_save_native_autotuning_results(core_context=core_context)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    info = det.get_cluster_info()
    hparams = info.trial.hparams
    distributed = det.core.DistributedContext.from_deepspeed()
    with det.core.init(distributed=distributed) as core_context:
        main(core_context, hparams)
