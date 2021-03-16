import inspect
from typing import Any, Dict, List, Sequence, Tuple, Union, cast

import pytorch_lightning as pl
import torch
from pytorch_lightning.trainer.optimizers import TrainerOptimizersMixin
from pytorch_lightning.utilities.model_helpers import is_overridden
from torch.optim.optimizer import Optimizer

from determined.errors import InvalidModelException
from determined.monkey_patch import monkey_patch
from determined.pytorch import LRScheduler, PyTorchCallback, PyTorchTrial, PyTorchTrialContext
from determined.tensorboard.metric_writers import pytorch
from determined.util import filter_duplicates, has_param
from determined_common import check

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


def check_compatibility(lm: pl.LightningModule) -> None:
    prefix = "Unsupported usage in LightningAdapter: "
    unsupported_members = {
        "backward",
        "get_progress_bar_dict",
        "manual_backward",
        "on_fit_end",
        "on_fit_start",
        "on_load_checkpoint",
        "on_pretrain_routine_end",
        "on_pretrain_routine_start",
        "on_save_checkpoint",
        "on_test_batch_end",
        "on_test_batch_start",
        "on_test_epoch_end",
        "on_test_epoch_start",
        "on_train_epoch_end",
        "optimizer_step",
        "optimizer_zero_grad",
        "setup",
        "tbptt_split_batch",
        "teardown",
        "test_dataloader",
        "test_epoch_end",
        "test_step",
        "test_step_end",
        "training_step_end",
        "transfer_batch_to_device",
        "validation_step_end",
    }

    members = inspect.getmembers(lm, predicate=inspect.ismethod)
    overridden_members = set(
        map(lambda m: m[0], filter(lambda m: is_overridden(m[0], lm), members))
    )

    matches = unsupported_members & overridden_members
    if len(matches) > 0:
        raise InvalidModelException(prefix + f"{matches}")

    for member in overridden_members:
        if has_param(getattr(lm, member), "dataloader_idx"):
            raise InvalidModelException(
                prefix
                + f'multiple dataloaders and `dataloader_idx` are not supported in "{member}"'
            )

    if has_param(lm.training_step, "hiddens", 4):
        raise InvalidModelException(prefix + '`hiddens` argument in "training_step"')

    if lm.trainer is not None:
        raise InvalidModelException(prefix + "Lightning Trainer")


def override_unsupported_nud(lm: pl.LightningModule, context: PyTorchTrialContext) -> None:
    writer = pytorch.TorchWriter()

    def lm_print(*args: Any, **kwargs: Any) -> None:
        if context.distributed.get_rank() == 0:
            print(*args, **kwargs)

    def lm_log_dict(a_dict: Dict, *args: Any, **kwargs: Any) -> None:
        if len(args) != 0 or len(kwargs) != 0:
            raise InvalidModelException(
                f"unsupported arguments to LightningModule.log {args} {kwargs}"
            )
        for metric, value in a_dict.items():
            if type(value) == int or type(value) == float:
                writer.add_scalar(metric, value, context.current_train_batch())

    def lm_log(name: str, value: Any, *args: Any, **kwargs: Any) -> None:
        lm_log_dict({name: value}, *args, **kwargs)

    lm.print = lm_print  # type: ignore
    lm.log = lm_log  # type: ignore
    lm.log_dict = lm_log_dict  # type: ignore


class _LightningAdapterState:
    def __init__(
        self, context: PyTorchTrialContext, lm: pl.LightningModule, optimizers: List[Optimizer]
    ):
        self.context = context
        self.lm = lm
        self.optimizers = optimizers


class LightningAdapter(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext, lightning_module: pl.LightningModule):
        check_compatibility(lightning_module)
        override_unsupported_nud(lightning_module, context)
        context.wrap_model(lightning_module)
        optimizers, lr_schedulers = self.setup_optimizers_schedulers(context, lightning_module)
        pls = _LightningAdapterState(context, lightning_module, optimizers)
        self._pls = pls

        # set lightning_module properties
        pls.lm.use_ddp = False  # type: ignore
        pls.lm.use_ddp2 = False  # type: ignore
        pls.lm.use_dp = False  # type: ignore
        pls.lm.use_tpu = False  # type: ignore
        type(pls.lm).local_rank = context.distributed.get_local_rank()  # type: ignore
        type(pls.lm).global_rank = context.distributed.get_rank()  # type: ignore
        pls.lm.use_amp = context.experimental._auto_amp or context._use_apex
        pls.lm.to(context.device)

    def build_callbacks(self) -> Dict[str, PyTorchCallback]:
        """
        build_callbacks defines a set of necessary PyTorchTrialCallback to support
        lightning. Override and merge the output of this build_callbacks with your
        desired callbacks.
        """
        context = self._pls.context
        lm = self._pls.lm

        class LightningAdapterCallback(PyTorchCallback):
            def on_training_epoch_start(self) -> None:
                if context._current_batch_idx is not None:
                    type(lm).current_epoch = context.current_train_epoch()  # type: ignore
                lm.on_train_epoch_start()

            def on_validation_epoch_start(self) -> None:
                lm.on_validation_epoch_start()

            def on_validation_epoch_end(self, outputs: List[Any]) -> None:
                lm.on_validation_epoch_end()
                lm.validation_epoch_end(outputs)

        return {"_lightning_module": LightningAdapterCallback()}

    def setup_optimizers_schedulers(
        self,
        context: PyTorchTrialContext,
        lightning_module: pl.LightningModule,
    ) -> Tuple[List[Optimizer], List[LRScheduler]]:
        """
        Wrap optimizers and lr_schedulers returned by `configure_optimizers` to
        work with Determined.
        Return: Wrapped `optimizers`, and `lr_schedulers` in a tuple
        """
        optimizers, lr_scheduler_dicts, opt_frequencies = TrainerOptimizersMixin().init_optimizers(
            lightning_module,
        )
        optimizers = cast(List[Optimizer], optimizers)
        lr_scheduler_dicts = cast(List[dict], lr_scheduler_dicts)

        def lightning_scheduler_dict_to_det(lrs: dict) -> LRScheduler:
            """
            input_dict = {
                'scheduler': None,
                'name': None,  # no custom name
                'interval': 'epoch',  # after epoch is over
                'frequency': 1,  # every epoch/batch
                'reduce_on_plateau': False,  # most often not ReduceLROnPlateau scheduler
                'monitor': monitor,  # value to monitor for ReduceLROnPlateau
                'strict': True,  # enforce that the monitor exists for ReduceLROnPlateau
            }
            """
            if lrs["reduce_on_plateau"]:
                raise InvalidModelException("LRScheduler reduce_on_plateaue is not supported")
            if lrs["monitor"] is not None:
                raise InvalidModelException("LRScheduler monitor is not supported")

            step_mode = (
                LRScheduler.StepMode.STEP_EVERY_EPOCH
                if lrs["interval"] == "epoch"
                else LRScheduler.StepMode.STEP_EVERY_BATCH
            )
            opt = getattr(lrs["scheduler"], "optimizer", None)
            if opt is not None:
                check.is_in(
                    opt,
                    self._pls.optimizers,
                    "Must use an optimizer that is returned by wrap_optimizer()",
                )
            return LRScheduler(lrs["scheduler"], step_mode, frequency=lrs["frequency"])

        optimizers = [context.wrap_optimizer(opt) for opt in optimizers]
        lr_schedulers = [lightning_scheduler_dict_to_det(lrs) for lrs in lr_scheduler_dicts]
        return optimizers, lr_schedulers

    def _build_train_args(self, batch: TorchData, batch_idx: int, opt_idx: int) -> List[Any]:
        # taken from pytorch_lightning
        args = [batch, batch_idx]

        if len(self._pls.optimizers) > 1:
            if has_param(self._pls.lm.training_step, "optimizer_idx"):
                args.append(opt_idx)
            else:
                num_opts = len(self._pls.optimizers)
                raise InvalidModelException(
                    f"Your LightningModule defines {num_opts} optimizers but "
                    f'training_step is missing the "optimizer_idx" argument.'
                )

        return args

    def train_batch(
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        type(self._pls.lm).global_step = batch_idx  # type: ignore
        self._pls.lm.on_train_batch_start(batch, batch_idx, dataloader_idx=0)

        Metric = Dict[str, Any]

        opt_metrics: List[Metric] = []
        metrics: Metric = {}

        for opt_idx, opt in enumerate(self._pls.optimizers):
            with monkey_patch(
                self._pls.lm, "optimizers", lambda *args, **kwargs: self._pls.optimizers
            ):
                self._pls.lm.toggle_optimizer(opt, opt_idx)
            train_args = self._build_train_args(batch, batch_idx, opt_idx)
            metrics = self._pls.lm.training_step(*train_args)  # type: ignore

            if metrics is None:
                continue
            elif not isinstance(metrics, dict):
                metrics = {"loss": metrics}

            self._pls.context.backward(metrics["loss"])
            self._pls.lm.on_after_backward()
            self._pls.context.step_optimizer(opt, auto_zero_grads=False)
            self._pls.lm.on_before_zero_grad(opt)
            opt.zero_grad()

            opt_metrics.append(metrics)
            with monkey_patch(
                self._pls.lm, "optimizers", lambda *args, **kwargs: self._pls.optimizers
            ):
                self._pls.lm.untoggle_optimizer(opt_idx)

        self._pls.lm.on_train_batch_end(metrics, batch, batch_idx, dataloader_idx=0)

        # report metrics accounting for duplicate metric names
        # across multiple optimizers
        metric_names: List[str] = []
        for opt_metric_dict in opt_metrics:
            metric_names += opt_metric_dict.keys()
        duplicate_metrics = filter_duplicates(metric_names)

        agg_metrics = {}
        for opt_idx, opt_metric_dict in enumerate(opt_metrics):
            for m_name, m_value in opt_metric_dict.items():
                if m_name in duplicate_metrics:
                    m_name = f"opt{opt_idx}_{m_name}"
                agg_metrics[m_name] = m_value
        return agg_metrics

    def evaluate_batch(self, batch: TorchData, batch_idx: int) -> Dict[str, Any]:
        self._pls.lm.on_validation_batch_start(batch, batch_idx, dataloader_idx=0)
        rv = self._pls.lm.validation_step(batch, batch_idx=batch_idx)  # type: ignore
        self._pls.lm.on_validation_batch_end(rv, batch, batch_idx, dataloader_idx=0)

        metrics = None
        if rv is None:
            metrics = {}
        elif not isinstance(rv, dict):
            metrics = {"loss": rv}
        else:
            metrics = rv
        return metrics
