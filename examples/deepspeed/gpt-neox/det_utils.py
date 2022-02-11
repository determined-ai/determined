from attrdict import AttrMap
import logging
import numpy as np

from megatron.neox_arguments import NeoXArgs
from eval_tasks.eval_adapter import run_eval_harness

from determined.pytorch import (
    PyTorchCallback,
    MetricReducer,
)
from determined.tensorboard.metric_writers.pytorch import TorchWriter


def calculate_batch_size(args):
    if "train_micro_batch_size_per_gpu" in args.overwrite_values:
        micro_cross_slot_batch_size = args.data_parallel_world_size * args.overwrite_values.train_micro_batch_size_per_gpu
        return (args.global_batch_size // micro_cross_slot_batch_size) * micro_cross_slot_batch_size
    return args.global_batch_size


def get_neox_args(context):
    args = AttrMap(context.get_hparams())
    exp_config = context.get_experiment_config()
    if args.search_world_size:
        args.overwrite_values["model_parallel_size"] = (
                context.distributed.get_size() //
                args.data_parallel_world_size //
                args.overwrite_values.pipe_parallel_size
        )
        print(args.overwrite_values)
    context.env._global_batch_size = calculate_batch_size(args)
    context.env._per_slot_batch_size = context.env.global_batch_size // args.data_parallel_world_size
    args.global_batch_size = context.env.global_batch_size
    print(context.get_global_batch_size(), context.get_per_slot_batch_size())

    # Gather overrides.
    overwrite_values = args.pop("overwrite_values")
    # We are going to overwrite certain neox_args with determined config values
    # from the experiment config to ensure consistency.
    assert (
            "batches" in exp_config["searcher"]["max_length"]
    ), "Please specify max_length in batches."
    assert (
            "batches" in exp_config["min_validation_period"]
    ), "Please specify min_validation_period in batches."
    overwrite_values.update(
        {
            "train_batch_size": args.global_batch_size,
            "train_iters": exp_config["searcher"]["max_length"]["batches"],
            "save_interval": exp_config["min_validation_period"]["batches"],
            "eval_interval": exp_config["min_validation_period"]["batches"],
            "global_num_gpus": context.distributed.get_size(),
            "seed": context.env.trial_seed,
        }
    )
    for k, v in overwrite_values.items():
        logging.info(f"Setting neox_args.{k} to {v}")

    # Build neox args.
    neox_args = NeoXArgs.process_parsed_deepy_args(
        args, overwrite_values=overwrite_values
    )
    assert args.data_parallel_world_size == int(
        neox_args.global_num_gpus
        / neox_args.model_parallel_size
        / neox_args.pipe_parallel_size
    ), "Data_parallel_world_size does not match number of replicas indicated by neox_args."
    return neox_args


class TensorboardWriter(PyTorchCallback):
    def __init__(self, writer: TorchWriter):
        self.tb_writer = writer.writer

    def on_validation_end(self, metrics):
        self.tb_writer.flush()

    def trial_cleanup(self) -> None:
        self.tb_writer.flush()
        self.tb_writer.close()


class EarlyStoppingCallback(PyTorchCallback):
    def __init__(self, trial):
        self.trial = trial

    def on_validation_start(self):
        if self.trial.reported_flops:
            self.trial.context.set_stop_requested(True)


class LMReducers(MetricReducer):
    def __init__(self, neox_args):
        self.char_level_ppl = neox_args.char_level_ppl
        self.token_count = 0
        self.char_count = 0
        self.lm_losses = []

    def update(self, lm_loss, token_count=None, char_count=None):
        self.lm_losses.append(lm_loss)
        if self.char_level_ppl:
            self.token_count += token_count
            self.char_count += char_count

    def reset(self):
        self.lm_losses = []
        self.token_count = 0
        self.char_count = 0

    def per_slot_reduce(self):
        return self.lm_losses, self.token_count, self.char_count

    def cross_slot_reduce(self, per_slot_metrics):
        lm_losses, token_count, char_count = zip(*per_slot_metrics)
        lm_losses = [item for sublist in lm_losses for item in sublist]

        metrics = {"lm_loss": np.mean(lm_losses)}
        metrics["lm_loss_ppl"] = np.exp(metrics["lm_loss"])
        if self.char_level_ppl:
            tokens_per_char = sum(token_count) / sum(char_count)
            metrics["lm_loss_char_lvl_ppl"] = np.exp(
                metrics["lm_loss"] * tokens_per_char
            )
        return metrics


class EvalHarness(PyTorchCallback):
    def __init__(self, model, forward_step_fn, neox_args):
        self.model = model
        self.forward_step_fn = forward_step_fn
        self.neox_args = neox_args

    def on_validation_end(self, metrics):
        # TODO: This hangs with pipeline parallel.
        metrics.update(
            run_eval_harness(
                self.model,
                self.forward_step_fn,
                self.neox_args,
                eval_tasks=self.neox_args.eval_tasks,
            )
        )
