import typing

from transformers import (
    TrainerCallback,
    TrainerState,
    TrainerControl,
    TrainingArguments,

)

from transformers.integrations import MLflowCallback, WandbCallback

import importlib.util
import os

import determined as det


class DetCallback(TrainerCallback):

    def __init__(self, core_context: det.core.Context, args: typing.Dict, filter_metrics: typing.List,
                 checkpoint_metadata: typing.Dict, tokenizer=None) -> None:
        super().__init__()

        self.core_context = core_context
        self.filter_metrics = filter_metrics
        self.checkpoint_metadata = checkpoint_metadata
        self.load_last_checkpoint(args)

    def on_log(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, model=None, logs=None,
               **kwargs):
        if state.is_world_process_zero:
            metric_type, metrics = self.process_log(logs)
            if metric_type == 'train':
                self.core_context.train.report_training_metrics(steps_completed=state.global_step, metrics=metrics)
            elif metric_type == 'eval' or metric_type == 'test':
                self.core_context.train.report_validation_metrics(steps_completed=state.global_step, metrics=metrics)
            else:
                pass

    def process_log(self, log):
        metric_type = self._metric_type(log)
        metrics = log

        if self.filter_metrics is not None:
            metrics = {}
            for k, v in log.items():
                if any(m in k for m in self.filter_metrics) is True:
                    metrics[k] = v

        return metric_type, metrics

    def _metric_type(self, d):
        for k, v in d.items():
            if k.startswith("eval"):
                return "eval"
            elif k.startswith("test"):
                return "test"
            else:
                return "train"

    def on_save(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, **kwargs):
        info = det.get_cluster_info()
        assert info is not None
        if state.is_world_process_zero:
            save_path = os.path.join(args.output_dir, f"checkpoint-{state.global_step}")
            ckpt_metadata = self.checkpoint_metadata
            ckpt_metadata["steps_completed"] = state.global_step
            ckpt_metadata["trial_id"] = info.trial.trial_id

            self.core_context.checkpoint.upload(save_path, ckpt_metadata)

    def load_last_checkpoint(self, args):
        info = det.get_cluster_info()
        assert info is not None
        latest_checkpoint = info.latest_checkpoint

        if latest_checkpoint is not None:
            metadata = self.core_context.checkpoint.get_metadata(latest_checkpoint)
            prev_trial_id = metadata["trial_id"]
            trial_id = info.trial.trial_id
            if trial_id != prev_trial_id:
                resume_step = 0
            else:
                resume_step = metadata['steps_completed']
            checkpoint_path = os.path.join(args.output_dir, f"checkpoint-{resume_step}")
            self.core_context.checkpoint.download(latest_checkpoint, checkpoint_path)

    def on_epoch_end(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, **kwargs):
        if self.core_context.preempt.should_preempt():
            # Terminate the process by returning from main.
            return


def is_determined_available():
    return importlib.util.find_spec("determined") is not None
