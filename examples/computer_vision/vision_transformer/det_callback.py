import typing

from transformers import (
    TrainerCallback,
    TrainerState,
    TrainerControl,
    TrainingArguments,
)

import os
import determined as det
import logging

import weakref
import importlib.util


class DetCallback(TrainerCallback):

    def __init__(self, args: typing.Dict, filter_metrics: typing.List = None,
                 tokenizer: typing.Any = None, tokenizer_options: typing.Dict = None,
                 checkpoint_metadata: typing.Dict = None) -> None:
        super().__init__()

        assert is_determined_available(), "DetCallback requires determined to be installed. Run `pip install determined`."
        import determined
        self._det = determined
        distributed = self._det.core.DistributedContext.from_torch_distributed()
        self.core_context = self._det.core.init(distributed=distributed)
        self.core_context.__enter__()
        weakref.finalize(self, exit_context, self.core_context)

        self.filter_metrics = filter_metrics
        self.user_checkpoint_metadata = checkpoint_metadata
        self.tokenizer = tokenizer
        self.tokenizer_options = tokenizer_options
        self.load_last_checkpoint(args)

        self.last_eval_metrics = None
        self.searcher_metric = self._det.get_cluster_info().trial._config['searcher']['metric']
        self.searcher_ops = self.core_context.searcher.operations()
        self.current_op = next(self.searcher_ops)

    def on_log(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, model=None, logs=None,
               **kwargs):
        if state.is_world_process_zero:
            metric_type, metrics = self.process_log(logs)
            if metric_type == 'train':
                self.core_context.train.report_training_metrics(steps_completed=state.global_step, metrics=metrics)
            elif metric_type == 'eval' or metric_type == 'test':
                self.core_context.train.report_validation_metrics(steps_completed=state.global_step, metrics=metrics)
                self.last_eval_metrics = metrics
            else:
                logging.warning(
                    f"Metrics not reported: unknown metric type in {logs}. Metrics should start with eval, test or train.")

    def process_log(self, log):
        metric_type = self._metric_type(log)
        metrics = log

        if self.filter_metrics:
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
        info = self._det.get_cluster_info()
        assert info is not None
        if state.is_world_process_zero:
            save_path = os.path.join(args.output_dir, f"checkpoint-{state.global_step}")
            det_checkpoint_metadata = {"steps_completed": state.global_step, "trial_id": info.trial.trial_id}

            if self.tokenizer_options is not None:
                det_checkpoint_metadata['tokenizer_options'] = self.tokenizer_options

            if self.tokenizer is not None:
                self.tokenizer.save_pretrained(os.path.join(save_path, "tokenizer"))

            if self.user_checkpoint_metadata is not None:
                self._on_save_user_data(save_path)

            self.core_context.checkpoint.upload(save_path, det_checkpoint_metadata)

    def _on_save_user_data(self, save_path):
        '''
        User-defined saving of objects from self.checkpoint_metadata under save_path.
        After objects are saved, Determined handles uploading and downloading objects to/from selected storage.
        '''
        logging.warning("No implementation for _on_save_user_data. "
                        "Objects passed to the callback via checkpoint_metadata will not be saved.")

    def load_last_checkpoint(self, args):
        info = self._det.get_cluster_info()
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
            return

        if state.epoch:
            if state.is_world_process_zero:
                self.current_op.report_progress(state.epoch)

            if round(state.epoch) >= self.current_op.length:
                if state.is_world_process_zero:
                    if self.searcher_metric not in self.last_eval_metrics:
                        logging.warning(
                            f"Searcher metric {self.searcher_metric} from the yaml config file does not match any "
                            f"of the evaluation metrics in {self.last_eval_metrics}. "
                            f"Proceeding with state.best_metric.")
                        self.current_op.report_completed(state.best_metric)
                    else:
                        self.current_op.report_completed(self.last_eval_metrics[self.searcher_metric])

                try:
                    self.current_op = next(self.searcher_ops)
                except StopIteration:
                    control.should_training_stop = True


def is_determined_available():
    return importlib.util.find_spec("determined") is not None


def exit_context(context):
    context.__exit__(None, None, None)


def override_training_args(training_args):
    hparams = det.get_cluster_info().trial.hparams
    for k, v in hparams.items():
        if hasattr(training_args, k):
            training_args.k = v
    return training_args
