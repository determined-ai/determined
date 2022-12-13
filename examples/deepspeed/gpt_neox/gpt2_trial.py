import logging
import pathlib
import traceback
from datetime import datetime

import megatron.training as megatron_train
import megatron.utils as megatron_utils
import torch
from attrdict import AttrMap
from det_utils import (
    EarlyStoppingCallback,
    EvalHarness,
    LMReducers,
    TensorboardWriter,
    get_neox_args,
)
from megatron import mpu
from megatron.checkpointing import load_checkpoint, save_checkpoint
from megatron.data.data_utils import build_datasets_from_neox_args

import deepspeed
from determined import LOG_FORMAT, InvalidHP
from determined.pytorch import DataLoader
from determined.pytorch.deepspeed import DeepSpeedTrial, DeepSpeedTrialContext, ModelParallelUnit
from determined.tensorboard.metric_writers.pytorch import TorchWriter

logging.basicConfig(level=logging.INFO, format=LOG_FORMAT)


class GPT2Trial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
        self.context = context
        self.exp_config = self.context.get_experiment_config()
        self.args = AttrMap(self.context.get_hparams())

        # Initalize and get arguments, timers, and Tensorboard writer.
        try:
            self.neox_args = get_neox_args(self.context)
        except:
            traceback.print_exc()
            raise InvalidHP("Could not parse neox_args.")
        logging.info(self.neox_args)
        self.wrapped_writer = TorchWriter()
        self.neox_args.tensorboard_writer = self.wrapped_writer.writer
        self.neox_args.configure_distributed_args()
        # The tokenizer needs to be built before model initialization in order to set the
        # required padded_vocab_size argument.
        self.neox_args.build_tokenizer()
        megatron_train.initialize_megatron(neox_args=self.neox_args)
        self.timers = megatron_utils.Timers(
            use_wandb=False, tensorboard_writer=self.neox_args.tensorboard_writer
        )

        # Model, optimizer, and learning rate.
        self.timers("model and optimizer").start()
        with deepspeed.zero.Init(enabled=self.neox_args.zero_optimization["stage"] == 3):
            (
                model,
                self.optimizer,
                self.lr_scheduler,
            ) = megatron_train.setup_model_and_optimizer(neox_args=self.neox_args)
        self.model = self.context.wrap_model_engine(model)
        self.context.set_mpu(
            ModelParallelUnit(
                mpu.get_data_parallel_rank(),
                mpu.get_data_parallel_world_size(),
                should_report_metrics=True,
                should_build_data_loader=self.should_build_data_loader(),
            )
        )
        self.timers("model and optimizer").stop()

        # Print setup timing.
        megatron_utils.print_rank_0("done with setups ...")
        self.timers.log(["model and optimizer"])
        megatron_utils.print_rank_0("training ...")

        # For tracking.
        if not self.args.search_world_size:
            self.reducer = self.context.wrap_reducer(
                LMReducers(self.neox_args), for_training=False, for_validation=True
            )
        self.report_memory_flag = True
        self.total_train_loss_dict = {}
        self.total_val_loss_dict = {}
        self.tflops = 0
        self.reported_flops = False
        self.overflow_monitor = megatron_utils.OverflowMonitor(self.optimizer)
        self.noise_scale_logger = megatron_utils.get_noise_scale_logger(self.neox_args)
        self.timers("interval time").start()

    def should_build_data_loader(self):
        if self.neox_args.is_pipe_parallel:
            is_first_stage = mpu.get_pipe_parallel_rank() == 0
            is_last_stage = mpu.get_pipe_parallel_rank() == mpu.get_pipe_parallel_world_size() - 1
            pipe_load = is_first_stage or is_last_stage
        else:
            pipe_load = True
        return mpu.get_model_parallel_rank() == 0 and pipe_load

    def build_callbacks(self):
        callbacks = {"tb": TensorboardWriter(self.wrapped_writer)}
        if self.neox_args.eval_tasks:
            callbacks["eval_tasks"] = EvalHarness(
                self.model, megatron_train.forward_step, self.neox_args
            )
        if self.args.search_world_size:
            callbacks["early_stopping"] = EarlyStoppingCallback(self)
        return callbacks

    def train_batch(self, data_iterator, epoch_idx, batch_idx):
        if self.neox_args.is_pipe_parallel:
            reduced_loss = megatron_train.train_step_pipe(
                neox_args=self.neox_args,
                timers=self.timers,
                model=self.model,
                data_iterator=data_iterator,
            )
        else:
            losses = []
            for _ in range(self.neox_args.gradient_accumulation_steps):
                self.timers("forward").start()
                loss = megatron_train.forward_step(
                    neox_args=self.neox_args,
                    timers=self.timers,
                    data_iterator=data_iterator,
                    model=self.model,
                )
                self.timers("forward").stop()
                losses.append(loss)
                # Calculate gradients, reduce across processes, and clip.
                self.timers("backward").start()
                megatron_train.backward_step(
                    neox_args=self.neox_args,
                    timers=self.timers,
                    optimizer=self.optimizer,
                    model=self.model,
                    loss=loss,
                )
                self.timers("backward").stop()
                # Update parameters.
                self.timers("optimizer").start()
                if self.neox_args.deepspeed:
                    self.model.step()
                else:
                    raise ValueError("Must be using deepspeed to run neox")
                self.timers("optimizer").stop()
            reduced_loss = {"lm_loss": megatron_utils.reduce_losses(losses).mean()}

        if self.neox_args.precision == "fp16" and self.model.optimizer.overflow:
            skipped_iter = 1
        else:
            skipped_iter = 0
        self.neox_args.iteration += 1

        self.overflow_monitor.check(skipped_iter)  # check for repeated overflow
        if self.neox_args.log_gradient_noise_scale:  # log noise scale if applicable
            self.noise_scale_logger.update()

        # get learning rate (if present) - if doing soft prompt tuning + pipe parallel, you
        # may have no tunable parameters on a specific rank
        if self.optimizer.param_groups:
            lr = self.optimizer.param_groups[0].get("lr", 0)
        else:
            lr = 0

        # Logging.
        self.report_memory_flag, additional_metrics = megatron_train.training_log(
            neox_args=self.neox_args,
            timers=self.timers,
            loss_dict=reduced_loss,
            total_loss_dict=self.total_train_loss_dict,
            learning_rate=lr,
            iteration=self.neox_args.iteration,
            loss_scale=self.optimizer.cur_scale if self.neox_args.precision == "fp16" else None,
            report_memory_flag=self.report_memory_flag,
            skipped_iter=skipped_iter,
            model=self.model,
            optimizer=self.optimizer,
            noise_scale_logger=self.noise_scale_logger,
            return_metrics=True,
        )
        if (
            additional_metrics is not None
            and additional_metrics["num_nans"] == 0
            and additional_metrics["num_skipped"] == 0
        ):
            self.tflops = additional_metrics["flops_per_sec_per_gpu"] / 10 ** 12

        if (
            self.neox_args.exit_interval
            and self.neox_args.iteration % self.neox_args.exit_interval == 0
        ):
            torch.distributed.barrier()
            time_str = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            megatron_utils.print_rank_0(
                "time: {} | exiting the program at iteration {}".format(
                    time_str, self.neox_args.iteration
                )
            )
            self.context.set_stop_requested(True)
        return reduced_loss

    def evaluate_batch(self, data_iterator, batch_idx):
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user defines evaluate_full_dataset().
        """
        if self.args.search_world_size:
            if self.tflops > 0:
                self.reported_flops = True
            return {"tflops": self.tflops}

        if data_iterator is not None:
            if self.neox_args.char_level_ppl:
                data_iterator = megatron_utils.CharCounter(data_iterator, self.neox_args.tokenizer)

        loss = megatron_train.forward_step(
            model=self.model,
            data_iterator=data_iterator,
            neox_args=self.neox_args,
            timers=self.timers,
        )

        if data_iterator is not None:
            if self.neox_args.char_level_ppl:
                self.reducer.update(
                    loss.item(), data_iterator.token_count, data_iterator.char_count
                )
            else:
                self.reducer.update(loss.item())

        if self.neox_args.deepspeed and self.neox_args.checkpoint_activations:
            deepspeed.checkpointing.reset()

        return {"lm_loss": loss}

    def build_training_data_loader(self):
        # Data stuff.
        self.timers("train/valid/test data dataset").start()
        (
            self.train_data,
            self.valid_data,
            self.test_data,
        ) = build_datasets_from_neox_args(self.neox_args)
        self.timers("train/valid/test data dataset").stop()
        self.timers.log(["train/valid/test data dataset"])
        return DataLoader(
            self.train_data,
            batch_size=self.neox_args.train_micro_batch_size_per_gpu,
            shuffle=True,
            num_workers=self.neox_args.num_workers,
            drop_last=True,
            pin_memory=False,
        )

    def build_validation_data_loader(self):
        return DataLoader(
            self.valid_data,
            batch_size=self.neox_args.train_micro_batch_size_per_gpu,
            num_workers=self.neox_args.num_workers,
            drop_last=True,
            pin_memory=False,
        )

    def save(self, context: DeepSpeedTrialContext, path: pathlib.Path) -> None:
        self.neox_args.save = str(path)
        save_checkpoint(
            neox_args=self.neox_args,
            iteration=self.neox_args.iteration,
            model=self.model,
            optimizer=self.optimizer,
            lr_scheduler=self.lr_scheduler,
        )

    def load(self, context: DeepSpeedTrialContext, path: pathlib.Path) -> None:
        self.neox_args.load = str(path)
        self.neox_args.iteration = load_checkpoint(
            neox_args=self.neox_args,
            model=self.model,
            optimizer=self.optimizer,
            lr_scheduler=self.lr_scheduler,
            inference=False,
        )
        megatron_utils.print_rank_0(
            f"Loading checkpoint and starting from iteration {self.neox_args.iteration}"
        )
