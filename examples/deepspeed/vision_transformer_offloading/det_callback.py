from transformers import (
    TrainerCallback,
    TrainerState,
    TrainerControl,
    TrainingArguments,
)

import determined as det
import os


class DetCallback(TrainerCallback):

    def __init__(self, core_context, tokenizer, metrics_names=['loss', 'accuracy']) -> None:
        super().__init__()
        self.core_context = core_context
        self.metrics_names = metrics_names
        # self.tokenizer_options = tokenizer_options
        self.tokenizer = tokenizer
        self.load_last_checkpoint()

    def on_log(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, model=None, logs=None,
               **kwargs):
        if state.is_world_process_zero:
            metric_type, metrics = self.process_log(logs)
            if metric_type == 'train':
                self.core_context.train.report_training_metrics(steps_completed=state.global_step, metrics=metrics)
            elif metric_type == 'eval':
                self.core_context.train.report_validation_metrics(steps_completed=state.global_step, metrics=metrics)
            else:
                # TODO: how to handle test metric?!
                raise RuntimeError('Panic')

    def process_log(self, log):
        metric_type = self._log_type(log)
        metrics = {}
        for k, v in log.items():
            if any(m in k for m in self.metrics_names) is True:
                metrics[k] = v
        return metric_type, metrics

    def _log_type(self, d):
        eval_prefix = "eval"
        test_prefix = "test"
        for k, v in d.items():
            if k.startswith(eval_prefix):
                return "eval"
            elif k.startswith(test_prefix):
                return "test"
            else:
                return "train"

    def on_save(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, **kwargs):
        print('Saving checkpoint')
        info = det.get_cluster_info()
        assert info is not None
        if state.is_world_process_zero:
            save_path = os.path.join(args.output_dir, f"checkpoint-{state.global_step}")
            checkpoint_metadata = {
                "steps_completed": state.global_step,
                "trial_id": info.trial.trial_id,
                # "tokenizer_options": self.tokenizer_options
            }

            self.tokenizer.save_pretrained(os.path.join(save_path, "tokenizer"))
            self.core_context.checkpoint.upload(save_path, checkpoint_metadata)

    # def on_epoch_end(self, args: TrainingArguments, state: TrainerState, control: TrainerControl, **kwargs):
    #     print('on epoch end')
    #     print('should be saving checkpoint now!')
    #
    def load_last_checkpoint(self):
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

            self.core_context.checkpoint.download(
                latest_checkpoint, os.path.join(self.checkpoint_dir, f"checkpoint-{resume_step}")
            )
