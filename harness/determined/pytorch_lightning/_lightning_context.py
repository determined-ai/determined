import logging
import pathlib
from typing import Any, Dict, Iterable, List, Optional, Union, cast

import pytorch_lightning as pl
import torch
from pytorch_lightning.loggers import LightningLoggerBase, TensorBoardLogger
from pytorch_lightning.plugins import Plugin
from pytorch_lightning.profiler import BaseProfiler
from torch.utils.data import DataLoader

import determined as det
from determined import errors, horovod
from determined import pytorch_lightning as dl
from determined import tensorboard, workload
from determined_common import check

VERY_LARGE_NUMBER = 9999999999999999
CHECKPOINT_FILE_NAME = "trainer.ckpt"


class LightningTrialContext(det.TrialContext):
    """Contains runtime information for ``PyTorch Lightning`` workflows."""

    def __init__(
        self,
        env: det.EnvContext,
        workloads: workload.Stream,
        load_path: Optional[pathlib.Path],
        rendezvous_info: det.RendezvousInfo,
        hvd_config: horovod.HorovodContext,
    ):
        super().__init__(env, workloads, load_path, rendezvous_info, hvd_config)

        self._workload_client = dl.WorkloadClient(self, self._checkpoint, self._validate)
        self._determined_callback = LightningWorkloadCallback(self, self._workload_client)
        self.trainer = None  # type: Optional[pl.Trainer]

    def init_trainer(
        self,
        logger: Union[LightningLoggerBase, Iterable[LightningLoggerBase], bool] = True,
        checkpoint_callback: bool = False,
        callbacks: Optional[Union[List[pl.Callback], pl.Callback]] = None,
        default_root_dir: Optional[str] = None,
        gradient_clip_val: float = 0,
        process_position: int = 0,
        log_gpu_memory: Optional[str] = None,
        progress_bar_refresh_rate: Optional[int] = 0,
        overfit_batches: Union[int, float] = 0.0,
        track_grad_norm: Union[int, float, str] = -1,
        check_val_every_n_epoch: int = 1,
        fast_dev_run: Union[int, bool] = False,
        accumulate_grad_batches: Union[int, Dict[int, int], List[list]] = 1,
        limit_train_batches: Union[int, float] = 1.0,
        limit_val_batches: Union[int, float] = 1.0,
        limit_test_batches: Union[int, float] = 1.0,
        limit_predict_batches: Union[int, float] = 1.0,
        val_check_interval: Union[int, float] = 1.0,
        flush_logs_every_n_steps: int = 100,
        log_every_n_steps: int = 50,
        sync_batchnorm: bool = False,
        precision: int = 32,
        weights_summary: Optional[str] = "top",
        weights_save_path: Optional[str] = None,
        num_sanity_val_steps: int = 2,
        resume_from_checkpoint: Optional[Union[pathlib.Path, str]] = None,
        profiler: Optional[Union[BaseProfiler, bool, str]] = None,
        benchmark: bool = False,
        deterministic: bool = False,
        reload_dataloaders_every_epoch: bool = False,
        auto_lr_find: Union[bool, str] = False,
        replace_sampler_ddp: bool = True,
        terminate_on_nan: bool = False,
        prepare_data_per_node: bool = True,
        plugins: Optional[Union[Plugin, str, list]] = None,
        amp_backend: str = "native",
        amp_level: str = "O2",
        automatic_optimization: Optional[bool] = None,
        move_metrics_to_cpu: bool = False,
        enable_pl_optimizer: Optional[bool] = None,  # will remove in v1.3
        multiple_trainloader_mode: str = "max_size_cycle",
        stochastic_weight_avg: bool = False,
        *args: List[Any],
        **kwargs: Dict[str, Any],
    ) -> None:
        """Initializes a trainer that uses the configuration merged from Trainer flags and
        the Determined :ref:`experiment configuration <experiment-configuration>`.

        .. note::

            When using Determined searcher or any validation policies that are configured
            in :ref:`experiment configuration <experiment-configuration>`, we will call
            ``Trainer.run_evaluation``. So all the validation-related Trainer flags will
            also apply to the validation loops that are initialized by Determined.

        How ``Trainer`` features are merged with the Determined
        :ref:`experiment configuration <experiment-configuration>`?

        -  The default logger is a ``TensorBoardLogger`` that writes the TensorBoard logs
           to a directory that will automatically be synced to the checkpoint storage
           configured in the :ref:`experiment configuration <checkpoint-storage>`.

        -  We schedule multi-GPU training tasks on computation resources automatically
           based on the :ref:`resource configuration <exp-config-resources>`.
           The flags ``num_nodes``, ``num_processes``, ``gpus``, ``auto_select_gpus``,
           ``tpu_cores``, and ``accelerator`` will be filled out automatically and cannot
           be configured by users. Note that we only support `Horovod
           <https://pytorch-lightning.readthedocs.io/en/1.2.0/advanced/multi_gpu.html#horovod>`_
           as the accelerator.

        -  We do not support the ``Trainer`` flags that control the length of training:
           ``max_epochs``, ``min_epochs``, ``max_steps``, and ``min_steps``. Users need
           to configure them in :ref:`searcher configuration <experiment-configuration_searcher>`.

        -  ``truncated_bptt_steps`` is not supported.

        See the `Pytorch Lightning Trainer documentation
        <https://pytorch-lightning.readthedocs.io/en/1.2.0/common/trainer.html>`_
        for the ``Trainer`` flags.
        """
        # Check disallowed arguments
        disallowed_args = [
            "num_nodes",
            "num_processes",
            "gpus",
            "auto_select_gpus",
            "tpu_cores",
            "accelerator",
            "max_epochs",
            "min_epochs",
            "max_steps",
            "min_steps",
            "truncated_bptt_steps",
        ]
        for arg in disallowed_args:
            check.is_none(getattr(args, arg, None), f"{arg} is not supported.")
            check.not_in(arg, kwargs, f"{arg} is not supported.")

        # Merge loggers.
        if logger is True:
            logger = TensorBoardLogger(
                save_dir=str(tensorboard.get_base_path({})),
                version="",
                name="",
            )

        # Merge callbacks.
        if not callbacks:
            callbacks = []
        elif isinstance(callbacks, pl.Callback):
            callbacks = [callbacks]
        callbacks.append(self._determined_callback)

        # Set up accelerator. We only support Horovod now.
        accelerator = "horovod" if self.hvd_config.use else None
        gpus = 1 if self.env.use_gpu else 0

        # Load checkpoint. If there is checkpoint to resume from,
        # then override the user-specified checkpoint.
        if self.load_path:
            resume_from_checkpoint = self._load_path()

        # Merge arguments that control the max length.
        max_length = self.get_experiment_config()["searcher"]["max_length"]
        records_per_epoch = self.get_experiment_config()["records_per_epoch"]
        if "records" in max_length:
            max_epochs = VERY_LARGE_NUMBER
            max_steps = max_length["records"] // self.get_global_batch_size()
            if records_per_epoch:
                max_epochs = (max_length["records"] + records_per_epoch - 1) // records_per_epoch
        elif "batches" in max_length:
            max_epochs = VERY_LARGE_NUMBER
            max_steps = max_length["batches"]
            if records_per_epoch:
                max_epochs = (
                    max_steps * self.get_global_batch_size() + records_per_epoch - 1
                ) // records_per_epoch
        elif "epochs" in max_length:
            max_epochs = max_length["epochs"]
            max_steps = None
        else:
            raise errors.InvalidConfigurationException(
                self.get_experiment_config(),
                "Experiment configuration must have searcher.max_length field",
            )

        self.trainer = pl.Trainer(
            logger=logger,
            checkpoint_callback=checkpoint_callback,
            callbacks=callbacks,
            default_root_dir=default_root_dir,
            gradient_clip_val=gradient_clip_val,
            process_position=process_position,
            num_nodes=self.distributed.get_num_agents(),
            num_processes=1,  # This is useful when `accelerator='ddp_cpu'`.
            gpus=gpus,
            auto_select_gpus=False,  # This should be False when using Horovod.
            tpu_cores=None,
            log_gpu_memory=log_gpu_memory,
            progress_bar_refresh_rate=progress_bar_refresh_rate,
            overfit_batches=overfit_batches,
            track_grad_norm=track_grad_norm,
            check_val_every_n_epoch=check_val_every_n_epoch,
            fast_dev_run=fast_dev_run,
            accumulate_grad_batches=accumulate_grad_batches,
            max_epochs=max_epochs,
            min_epochs=0,  # This is not set in favor of early stopping anytime.
            max_steps=max_steps,
            min_steps=0,  # This is not set in favor of early stopping anytime.
            limit_train_batches=limit_train_batches,
            limit_val_batches=limit_val_batches,
            limit_test_batches=limit_test_batches,
            limit_predict_batches=limit_predict_batches,
            val_check_interval=val_check_interval,
            flush_logs_every_n_steps=flush_logs_every_n_steps,
            log_every_n_steps=log_every_n_steps,
            accelerator=accelerator,
            sync_batchnorm=sync_batchnorm,
            precision=precision,
            weights_summary=weights_summary,
            weights_save_path=weights_save_path,
            num_sanity_val_steps=num_sanity_val_steps,
            resume_from_checkpoint=resume_from_checkpoint,
            profiler=profiler,
            benchmark=benchmark,
            deterministic=deterministic,
            reload_dataloaders_every_epoch=reload_dataloaders_every_epoch,
            auto_lr_find=auto_lr_find,
            replace_sampler_ddp=replace_sampler_ddp,
            terminate_on_nan=terminate_on_nan,
            prepare_data_per_node=prepare_data_per_node,
            plugins=plugins,
            amp_backend=amp_backend,
            amp_level=amp_level,
            automatic_optimization=automatic_optimization,
            move_metrics_to_cpu=move_metrics_to_cpu,
            enable_pl_optimizer=enable_pl_optimizer,
            multiple_trainloader_mode=multiple_trainloader_mode,
            stochastic_weight_avg=stochastic_weight_avg,
            *args,
            **kwargs,
        )

        def fail_trainer_tune(*args: Any, **kwargs: Any) -> None:
            raise errors.InvalidTrialException("Trainer.tune is supported in LightningTrial.")

        self.trainer.tune = fail_trainer_tune  # type: ignore

    def _load_path(self) -> str:
        return str(cast(pathlib.Path, self.load_path).joinpath(CHECKPOINT_FILE_NAME))

    def _check_model_def(self) -> None:
        check.is_not_none(
            self.trainer,
            "Must call LightningTrialContext.init_trainer in the LightningTrial.__init__",
        )

    def _checkpoint(self, path: pathlib.Path) -> None:
        if not self.distributed.is_chief():
            self._workload_client.finish_checkpoint(workload.Skipped())
            return

        self.trainer = cast(pl.Trainer, self.trainer)
        self.trainer.save_checkpoint(path.joinpath(CHECKPOINT_FILE_NAME))

        self._workload_client.finish_checkpoint(
            {
                "framework": f"pytorch-lightning-{pl.__version__}",
                "format": "",
            }
        )

    def _validate(self) -> None:
        self.trainer = cast(pl.Trainer, self.trainer)
        self.trainer.run_evaluation()  # type: ignore

        self._workload_client.finish_validate()

    def fit(
        self,
        model: Optional[pl.LightningModule] = None,
        train_dataloader: Optional[DataLoader] = None,
        val_dataloaders: Optional[Union[DataLoader, List[DataLoader]]] = None,
        datamodule: Optional[pl.LightningDataModule] = None,
    ) -> None:
        """Runs the Trainer fitting loop.

        See `Trainer.fit
        <https://pytorch-lightning.readthedocs.io/en/1.2.0/common/trainer.html#fit>`_
        for more information.
        """
        self._check_model_def()
        self.trainer = cast(pl.Trainer, self.trainer)
        self.model = cast(pl.LightningModule, model)

        with self._workload_client.enter_training():
            self.trainer.fit(self.model, train_dataloader, val_dataloaders, datamodule)


class LightningWorkloadCallback(pl.Callback):
    def __init__(
        self,
        context: LightningTrialContext,
        workload_client: dl.WorkloadClient,
    ):
        self.context = context
        self.workload_client = workload_client

    def on_pretrain_routine_end(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        # HACK: reset current_epoch and global_step here after loading checkpoints
        # that contains current_epoch or global_step so that the trainer states
        # are consistent with the trial workloads.
        self.context.trainer = cast(pl.Trainer, self.context.trainer)
        self.context.trainer.current_epoch = 0
        self.context.trainer.global_step = self.context._cur_total_batches
        logging.info(
            f"Resetting Trainer: current_epoch={self.context.trainer.current_epoch}, "
            f"global_step={self.context.trainer.global_step}"
        )

    def on_train_start(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        logging.info("Trainer is training")
        self._print_trainer_states()

    def on_train_batch_start(
        self,
        trainer: pl.Trainer,
        pl_module: pl.LightningModule,
        batch: Any,
        batch_idx: int,
        dataloader_idx: int,
    ) -> None:
        self.context._cur_total_batches += 1

    def on_train_batch_end(
        self,
        trainer: pl.Trainer,
        pl_module: pl.LightningModule,
        outputs: Any,
        batch: Any,
        batch_idx: int,
        dataloader_idx: int,
    ) -> None:
        batch_metrics = self._process_train_outputs_in_local_slot(outputs)
        self.workload_client.finish_train_batch(batch_metrics)

    def on_validation_start(self, trainer: pl.Trainer, pl_module: pl.LightningModule) -> None:
        logging.info("Trainer is validating")
        self._print_trainer_states()

    def on_validation_batch_end(
        self,
        trainer: pl.Trainer,
        pl_module: pl.LightningModule,
        outputs: Any,
        batch: Any,
        batch_idx: int,
        dataloader_idx: int,
    ) -> None:
        batch_metrics = self._process_val_outputs_in_local_slot(outputs)
        self.workload_client.finish_validate_batch(batch_metrics)

    def _print_trainer_states(self) -> None:
        trainer = cast(pl.Trainer, self.context.trainer)
        logging.info(
            f"Trainer: state={trainer._running_stage}, "
            f"current_epoch={trainer.current_epoch}, global_step={trainer.global_step}, "
            f"local_rank={trainer.local_rank}, global_rank={trainer.global_rank}, "
            f"node_rank={trainer.node_rank}, num_nodes={trainer.num_nodes}, "
            f"num_gpus={trainer.num_gpus}, on_gpu={trainer.on_gpu}, "
            f"log_dir={trainer.log_dir}, default_root_dir={trainer.default_root_dir}"
        )

    @staticmethod
    def _evict_special_keys_from_pl_outs(metrics: Dict[str, Any]) -> Dict[str, Any]:
        if "log" in metrics:
            metrics.pop("log")
        if "progress_bar" in metrics:
            metrics.pop("progress_bar")
        return metrics

    @staticmethod
    def _detach_metrics(metrics: Dict[str, Any]) -> Dict[str, Any]:
        for name, metric in metrics.items():
            if isinstance(metric, torch.Tensor):
                metric = metric.cpu().detach().numpy()
            metrics[name] = metric
        return metrics

    @staticmethod
    def _process_train_outputs_in_local_slot(outputs: Any) -> Dict[str, Any]:
        # outputs come from pl.Trainer.process_train_step_outputs(...). The training step
        # outputs a list per optimizer. The list contains the outputs at each time step.
        # When no TBPTT is used, then the list has 1 item per batch; when TBPTT is used,
        # then the list has n items (1 per time step).
        # An example output of two optimizers is
        # [    [    {'loss': tensor(0.6935, device='cuda:0')}    ],
        #      [    {'loss': tensor(0.6926, device='cuda:0')}    ]    ]
        # Reference:
        # https://github.com/PyTorchLightning/pytorch-lightning/blob/
        # 40d5a9d6df4c5102f79ec67ccc7daae21654dc96/pytorch_lightning/trainer/training_loop.py#L859
        check.is_instance(outputs, list, "outputs must be a list of optimizer outputs.")

        merged_outs = {}
        for opt_idx, opt_outs in enumerate(outputs):
            check.is_instance(opt_outs, list, "optimizer outputs must be a list.")

            # TODO: support tbptt metrics reducing.
            check.len_eq(opt_outs, 1, "tbptt is not supported.")

            opt_out = LightningWorkloadCallback._evict_special_keys_from_pl_outs(opt_outs[0])
            opt_out = LightningWorkloadCallback._detach_metrics(opt_out)
            opt_out = {f"opt{opt_idx}_{k}": opt_out[k] for k in opt_out}

            merged_outs.update(opt_out)
        return merged_outs

    @staticmethod
    def _process_val_outputs_in_local_slot(outputs: Any) -> Dict[str, Any]:
        res = LightningWorkloadCallback._evict_special_keys_from_pl_outs(outputs)
        res = LightningWorkloadCallback._detach_metrics(res)
        return res
