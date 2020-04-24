"""
This file contains the code necessary to run the Tensorpack RCNN model inside Determined. The structure
of things is adapted for Determined, but much of the actual logic is taken pretty directly from the
original example files.
"""

import json
import os
import time
from typing import Any, Dict, List, Optional, Tuple, Union, cast

import tensorflow as tf

import tensorpack as tp
from config import config as cfg
from config import finalize_configs
from dataset import register_coco
from determined.tensorpack import (
    Evaluator,
    SchedulePoint,
    ScheduleSetter,
    TensorpackTrial,
)

from data import DatasetRegistry, get_eval_dataflow, get_train_dataflow
from eval import multithread_predict_dataflow, predict_dataflow
from modeling.generalized_rcnn import ResNetFPNModel

try:
    import horovod.tensorflow as hvd
except ImportError:
    pass


class RCNNEvaluator(Evaluator):  # type: ignore
    """
    The class that contains the code to compute evaluation metrics. This is essentially duplicating
    much of Tensorpack's `EvalCallback` class, but that class doesn't actually return the metrics,
    so we can't use it directly.
    """

    def __init__(
        self,
        eval_dataset: str,
        in_names: List[str],
        out_names: List[str],
        machine_rank: int,
        num_gpus: int,
        trainer_type: str,
        is_aws: bool,
        is_gcs: bool,
        output_dir: str = "/tmp",
    ):
        self._eval_dataset = eval_dataset
        self._in_names, self._out_names = in_names, out_names
        self._output_dir = output_dir
        self.machine_rank = machine_rank
        self.num_gpus = num_gpus
        self.trainer_type = trainer_type
        self.is_aws = is_aws
        self.is_gcs = is_gcs

    def set_up_graph(self, trainer: tp.Trainer) -> None:
        self.trainer = trainer
        if self.trainer_type == "replicated":
            # Use multiple predictor threads per GPU to get better throughput.
            self.num_predictor = self.num_gpus * 2
            self.predictors = [
                self._build_predictor(k % self.num_gpus) for k in range(self.num_predictor)
            ]
            self.dataflows = [
                get_eval_dataflow(  # type: ignore
                    self._eval_dataset,
                    self.is_aws,
                    self.is_gcs,
                    shard=k,
                    num_shards=self.num_predictor,
                )
                for k in range(self.num_predictor)
            ]
        else:
            if self.machine_rank == 0:
                # Run validation on one machine.
                self.predictor = self._build_predictor(0)
                self.dataflow = get_eval_dataflow(
                    self._eval_dataset,
                    self.is_aws,
                    self.is_gcs,
                    shard=hvd.local_rank(),
                    num_shards=hvd.local_size(),
                )

            # All workers must take part in this barrier, even if they
            # are not performing validation.
            self.barrier = hvd.allreduce(tf.random_normal(shape=[1]))

    def _build_predictor(self, idx: int) -> Any:
        return self.trainer.get_predictor(self._in_names, self._out_names, device=idx)

    def compute_validation_metrics(self) -> Any:
        if self.trainer_type == "replicated":
            all_results = multithread_predict_dataflow(
                self.dataflows, self.predictors
            )  # type: ignore
        else:
            filenames = [
                os.path.join(
                    self._output_dir, "outputs{}-part{}.json".format(self.trainer.global_step, rank)
                )
                for rank in range(hvd.local_size())
            ]

            if self.machine_rank == 0:
                local_results = predict_dataflow(self.dataflow, self.predictor)
                fname = filenames[hvd.local_rank()]
                with open(fname, "w") as f:
                    json.dump(local_results, f)
            self.barrier.eval()
            if hvd.rank() > 0:
                return
            all_results = []
            for fname in filenames:
                with open(fname, "r") as f:
                    obj = json.load(f)
                all_results.extend(obj)

        output_file = os.path.join(
            self._output_dir,
            "{}-outputs{}-{}.json".format(
                self._eval_dataset, self.trainer.global_step, time.time()
            ),
        )

        metrics = DatasetRegistry.get(self._eval_dataset).eval_inference_results(  # type: ignore
            all_results, output_file
        )

        # If there are no detections, the metrics result is totally empty, instead of containing
        # zeroes. Ensure that the main evaluation metric has some value.
        metrics.setdefault("mAP(bbox)/IoU=0.5:0.95", 0)

        return metrics


def make_schedule(num_gpus: int) -> List[SchedulePoint]:
    """
    This takes the original logic for setting the learning rate and expresses the resulting schedule
    in a more generic way.
    """
    factor = 8 / num_gpus

    init_lr = cfg.TRAIN.WARMUP_INIT_LR * min(factor, 1.0)
    schedule = [
        SchedulePoint(0, init_lr, interp="linear"),
        SchedulePoint(cfg.TRAIN.WARMUP, cfg.TRAIN.BASE_LR),
    ]

    for idx, steps in enumerate(cfg.TRAIN.LR_SCHEDULE[:-1]):
        mult = 0.1 ** (idx + 1)
        schedule.append(SchedulePoint(round(steps * factor), cfg.TRAIN.BASE_LR * mult))

    return schedule


class DeterminedResNetFPNModel(ResNetFPNModel):  # type: ignore
    """
    Determined assumes that the loss tensor is called 'loss', but the RCNN code calls its final tensor
    'total_cost'; this class aliases it.
    """

    def build_graph(self, *inputs: Any) -> Any:
        loss = super().build_graph(*inputs)
        if loss is not None:
            loss = tf.identity(loss, name="loss")
        return loss


class RCNNTrial(TensorpackTrial):  # type: ignore
    def __init__(self, context: Any) -> None:
        self.context = context
        self.trainer_type = None  # type: Optional[str]
        register_coco(  # type: ignore
            cfg.DATA.BASEDIR, self.context.get_hparam("is_aws"), self.context.get_hparam("is_gcs"),
        )

    def build_model(self, trainer_type: str) -> tp.ModelDesc:
        cfg.DATA.NUM_WORKERS = self.context.get_hparam("num_workers")
        cfg.MODE_MASK = True
        cfg.MODE_FPN = True
        if not self.context.get_hparam("is_gcs"):
            cfg.DATA.BASEDIR = "/rcnn-data/COCO/DIR"
        cfg.TRAIN.LR_SCHEDULE = [240000, 320000, 360000]  # "2x" schedule in Detectron.
        cfg.TRAIN.BASE_LR = 1e-2 * self.context.get_experiment_config().get("optimizations").get(
            "aggregation_frequency"
        )
        cfg.TRAIN.WARMUP = self.context.get_hparam("warmup_iterations")
        cfg.TRAIN.GRADIENT_CLIP = self.context.get_hparam("gradient_clipping")
        cfg.TRAINER = trainer_type
        self.trainer_type = trainer_type

        finalize_configs(is_training=True)  # type: ignore

        return DeterminedResNetFPNModel()

    def batch_size(self) -> int:
        return cast(int, self.context.get_per_slot_batch_size())

    def build_training_dataflow(self) -> tp.DataFlow:
        return get_train_dataflow(
            self.context.get_hparam("is_aws"), self.context.get_hparam("is_gcs")
        )

    def training_metrics(self) -> List[str]:
        return ["learning_rate"]

    def validation_metrics(self) -> Union[List[str], Evaluator]:
        assert self.trainer_type
        num_gpus_per_agent = (
            self.context.distributed.get_size() // self.context.distributed.get_num_agents()
        )
        machine_rank = self.context.distributed.get_rank() // num_gpus_per_agent
        return RCNNEvaluator(
            eval_dataset="coco_minival2014",
            in_names=["image"],
            out_names=["output/boxes", "output/scores", "output/labels", "output/masks"],
            machine_rank=machine_rank,
            num_gpus=num_gpus_per_agent,
            trainer_type=self.trainer_type,
            is_aws=self.context.get_hparam("is_aws"),
            is_gcs=self.context.get_hparam("is_gcs"),
        )

    def tensorpack_callbacks(self) -> List[tp.Callback]:
        num_workers = max(1, self.context.distributed.get_num_agents())
        return [ScheduleSetter("learning_rate", make_schedule(self.context.distributed.get_size()))]

    def load_backbone_weights(self) -> Optional[str]:
        # TODO: This is a temporary way of specifying the backbone weights to load when training
        # from scratch; our API for handling this case isn't complete.
        backbone_weights_load_path = "/root/ImageNet-R50-AlignPadding.npz"
        return backbone_weights_load_path
