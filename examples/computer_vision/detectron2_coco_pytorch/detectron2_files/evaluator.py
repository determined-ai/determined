"""
Slightly altered version of Detectron2's evaluator.py

Added EvaluatorReducer for Determined Ai

To create a new DatasetEvaluator class must include:
    evaluator_name: str
    setup_data(values): takes output from cross_slot_reduce and formats for evaluate()
"""

# Copyright (c) Facebook, Inc. and its affiliates. All Rights Reserved
import datetime
import logging
import os
import time
from collections import OrderedDict
from contextlib import contextmanager

import torch
from detectron2.data import MetadataCatalog
from detectron2.utils.comm import get_world_size, is_main_process
from detectron2.utils.logger import log_every_n_seconds
from detectron2_files.coco_evaluation import COCOEvaluator
from detectron2_files.panoptic_evaluation import COCOPanopticEvaluator
from detectron2_files.sem_seg_evaluation import SemSegEvaluator

from determined.pytorch import MetricReducer


class DatasetEvaluator:
    """
    Base class for a dataset evaluator.
    The function :func:`inference_on_dataset` runs the model over
    all samples in the dataset, and have a DatasetEvaluator to process the inputs/outputs.
    This class will accumulate information of the inputs/outputs (by :meth:`process`),
    and produce evaluation results in the end (by :meth:`evaluate`).
    """

    def reset(self):
        """
        Preparation for a new round of evaluation.
        Should be called before starting a round of evaluation.
        """
        pass

    def process(self, inputs, outputs):
        """
        Process the pair of inputs and outputs.
        If they contain batches, the pairs can be consumed one-by-one using `zip`:
        .. code-block:: python
            for input_, output in zip(inputs, outputs):
                # do evaluation on single input/output pair
                ...
        Args:
            inputs (list): the inputs that's used to call the model.
            outputs (list): the return value of `model(inputs)`
        """
        pass

    def evaluate(self):
        """
        Evaluate/summarize the performance, after processing all input/output pairs.
        Returns:
            dict:
                A new evaluator class can return a dict of arbitrary format
                as long as the user can process the results.
                In our train_net.py, we expect the following format:
                * key: the name of the task (e.g., bbox)
                * value: a dict of {metric name: score}, e.g.: {"AP50": 80}
        """
        pass


class DatasetEvaluators(DatasetEvaluator):
    """
    Updated to work with Determined

    Wrapper class to combine multiple :class:`DatasetEvaluator` instances.
    This class dispatches every evaluation call to
    all of its :class:`DatasetEvaluator`.
    """

    def __init__(self, evaluators):
        """
        Args:
            evaluators (list): the evaluators to combine.
        """
        super().__init__()
        self._evaluators = evaluators

    def reset(self):
        for evaluator in self._evaluators:
            evaluator.reset()

    def process(self, inputs, outputs):
        values = {}
        for evaluator in self._evaluators:
            preds = evaluator.process(inputs, outputs)
            values.update(preds)
        return values

    def setup_data(self, values):
        for evaluator in self._evaluators:
            evaluator_name = evaluator.evaluator_name
            evaluator.setup_data(values)

    def evaluate(self):
        results = OrderedDict()
        for evaluator in self._evaluators:
            result = evaluator.evaluate()
            for k, v in result.items():
                assert (
                    k not in results
                ), "Different evaluators produce results with the same key {}".format(k)
                results[k] = v
        return results


def inference_on_dataset(model, data_loader, evaluator):
    """
    Run model on the data_loader and evaluate the metrics with evaluator.
    Also benchmark the inference speed of `model.forward` accurately.
    The model will be used in eval mode.
    Args:
        model (nn.Module): a module which accepts an object from
            `data_loader` and returns some outputs. It will be temporarily set to `eval` mode.
            If you wish to evaluate a model in `training` mode instead, you can
            wrap the given model and override its behavior of `.eval()` and `.train()`.
        data_loader: an iterable object with a length.
            The elements it generates will be the inputs to the model.
        evaluator (DatasetEvaluator): the evaluator to run. Use `None` if you only want
            to benchmark, but don't want to do any evaluation.
    Returns:
        The return value of `evaluator.evaluate()`
    """
    num_devices = get_world_size()
    logger = logging.getLogger(__name__)
    logger.info("Start inference on {} images".format(len(data_loader)))

    total = len(data_loader)  # inference data loader must have a fixed length
    if evaluator is None:
        # create a no-op evaluator
        evaluator = DatasetEvaluators([])
    evaluator.reset()

    num_warmup = min(5, total - 1)
    start_time = time.perf_counter()
    total_compute_time = 0
    with inference_context(model), torch.no_grad():
        for idx, inputs in enumerate(data_loader):
            if idx == num_warmup:
                start_time = time.perf_counter()
                total_compute_time = 0

            start_compute_time = time.perf_counter()
            outputs = model(inputs)
            if torch.cuda.is_available():
                torch.cuda.synchronize()
            total_compute_time += time.perf_counter() - start_compute_time
            evaluator.process(inputs, outputs)

            iters_after_start = idx + 1 - num_warmup * int(idx >= num_warmup)
            seconds_per_img = total_compute_time / iters_after_start
            if idx >= num_warmup * 2 or seconds_per_img > 5:
                total_seconds_per_img = (time.perf_counter() - start_time) / iters_after_start
                eta = datetime.timedelta(seconds=int(total_seconds_per_img * (total - idx - 1)))
                log_every_n_seconds(
                    logging.INFO,
                    "Inference done {}/{}. {:.4f} s / img. ETA={}".format(
                        idx + 1, total, seconds_per_img, str(eta)
                    ),
                    n=5,
                )

    # Measure the time only for this worker (before the synchronization barrier)
    total_time = time.perf_counter() - start_time
    total_time_str = str(datetime.timedelta(seconds=total_time))
    # NOTE this format is parsed by grep
    logger.info(
        "Total inference time: {} ({:.6f} s / img per device, on {} devices)".format(
            total_time_str, total_time / (total - num_warmup), num_devices
        )
    )
    total_compute_time_str = str(datetime.timedelta(seconds=int(total_compute_time)))
    logger.info(
        "Total inference pure compute time: {} ({:.6f} s / img per device, on {} devices)".format(
            total_compute_time_str, total_compute_time / (total - num_warmup), num_devices
        )
    )

    results = evaluator.evaluate()
    # An evaluator may return None when not in main process.
    # Replace it by an empty dict instead to make it easier for downstream code to handle
    if results is None:
        results = {}
    return results


@contextmanager
def inference_context(model):
    """
    A context where the model is temporarily changed to eval mode,
    and restored to previous mode afterwards.
    Args:
        model: a torch Module
    """
    training_mode = model.training
    model.eval()
    yield
    model.train(training_mode)


class EvaluatorReducer(MetricReducer):
    """
    Reducer class used for Determined.

    There can be 1 or N number of evaluators
    """

    def __init__(self, evaluators):
        self.reset()
        self.evaluators = evaluators

    def reset(self):
        self.values = {}

    def update(self, values):
        for key in values.keys():
            # add key if haven't seen evaluator
            if key not in self.values:
                self.values[key] = []

            # If the values are just a list, combine into 1 array
            # This is most evaluators
            if isinstance(values[key], list):
                self.values[key].extend(values[key])

            # For evaluators that have a list and bin count
            elif isinstance(values[key], dict):
                for key2 in values[key].keys():
                    if isinstance(values[key], list):
                        self.values[key][key2].extend(values[key][key2])
                    else:
                        # np.array bin count
                        # the bins will be updated in the evaluators as a class variable
                        # so it will have the most up to date per slot
                        self.values[key][key2] = values[key][key2]

    def per_slot_reduce(self):
        # Because the chosen update() mechanism is so
        # efficient, this is basically a noop.
        return self.values

    def cross_slot_reduce(self, per_slot_metrics):
        val = {}
        # Should loop based on N gpu
        for gpu_slot in per_slot_metrics:
            # Typically will loop once unless using multiple evaluators
            for key, values in gpu_slot.items():  # eval name, values
                if key not in val:
                    val[key] = []

                # If the values are just a list, combine into 1 array
                # This is most evaluators
                if isinstance(values, list):
                    val[key].extend(values)

                # For evaluators that have a list and bin count
                elif isinstance(values, dict):
                    for key2 in values.keys():  # Should have 2 keys
                        if isinstance(values[key2], list):
                            val[key][key2].extend(values[key2])
                        else:
                            # np.array bin count. The setup_data will combine bins
                            val[key][key2].append(values[key2])

        self.evaluators.setup_data(val)
        results = self.evaluators.evaluate()

        results = self.parse_results(results)
        return results

    def parse_results(self, results):
        fmt_results = {}
        for result_keys in results:
            for key, value in results[result_keys].items():
                new_key = result_keys + key
                fmt_results[new_key] = value
        return fmt_results


def get_evaluator(cfg, dataset_name, output_folder=None, fake=False):
    """
    This is a slightly altered detectron2 function. Below are original comments:

    Create evaluator(s) for a given dataset.
    This uses the special metadata "evaluator_type" associated with each builtin dataset.
    For your own dataset, you can simply create an evaluator manually in your
    script and do not have to worry about the hacky if-else logic here.
    """
    if output_folder is not None:
        output_folder = os.path.join(output_folder, "inference")

    evaluator_list = []
    evaluator_type = MetadataCatalog.get(dataset_name).evaluator_type
    if evaluator_type in ["sem_seg", "coco_panoptic_seg"]:
        evaluator_list.append(
            SemSegEvaluator(
                dataset_name,
                distributed=True,
                num_classes=cfg.MODEL.SEM_SEG_HEAD.NUM_CLASSES,
                ignore_label=cfg.MODEL.SEM_SEG_HEAD.IGNORE_VALUE,
                output_dir=output_folder,
            )
        )
    if evaluator_type in ["coco", "coco_panoptic_seg"]:
        evaluator_list.append(COCOEvaluator(dataset_name, cfg, True, output_folder, fake))
    if evaluator_type == "coco_panoptic_seg":
        evaluator_list.append(COCOPanopticEvaluator(dataset_name, output_folder))
    if len(evaluator_list) == 0:
        raise NotImplementedError(
            "no Evaluator for the dataset {} with the type {}".format(dataset_name, evaluator_type)
        )
    if len(evaluator_list) == 1:
        return evaluator_list[0]
    return DatasetEvaluators(evaluator_list)
