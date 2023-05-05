# type: ignore
"""
A one-variable linear model with no bias. The dataset emits only pairs of (data, label) = (1, 1),
meaning that the one weight in the model should approach 1 as gradient descent continues.

We will use the mean squared error as the loss.  Since each record is the same, the "mean" part of
mean squared error means we can analyze every batch as if were just one record.

Now, we can calculate the mean squared error to ensure that we are getting the gradient we are
expecting.

let:
    R = learning rate (constant)
    l = loss
    w0 = the starting value of the one weight
    w' = the updated value of the one weight

then calculate the loss:

(1)     l = (label - (data * w0)) ** 2

take derivative of loss WRT w

(2)     dl/dw = - 2 * data * (label - (data * w0))

gradient update:

(3)     update = -R * dl/dw = 2 * R * data * (label - (data * w0))

Finally, we can calculate the updated weight (w') in terms of w0:

(4)     w' = w0 + update = w0 + 2 * R * data * (label - (data * w0))
"""

import logging
from typing import Any, Dict, Iterable, List, Optional, Tuple, cast

import numpy as np
import torch

from determined import experimental, pytorch
from determined.common import yaml
from determined.pytorch import samplers
from tests.experiment.fixtures import pytorch_counter_callback

try:
    import apex
except ImportError:  # pragma: no cover
    pass


class OnesDataset(torch.utils.data.Dataset):
    def __len__(self) -> int:
        return 64

    def __getitem__(self, index: int) -> Tuple:
        return torch.Tensor([float(1)]), torch.Tensor([float(1)])


class TriangleLabelSum(pytorch.MetricReducer):
    """Return a sum of (label_sum * batch_index) for every batch (labels are always 1 here)."""

    @staticmethod
    def expect(batch_size, idx_start, idx_end):
        """What to expect during testing."""
        return sum(batch_size * idx for idx in range(idx_start, idx_end))

    def __init__(self) -> None:
        self.reset()

    def reset(self) -> None:
        self.sum = 0
        # We don't actually expose a batch_idx for evaluation, so we track the number of batches
        # since the last reset(), which is only accurate during evaluation workloads or the very
        # first training workload.
        self.count = 0

    def update(self, label_sum: torch.Tensor, batch_idx: Optional[int]) -> None:
        self.sum += label_sum * (batch_idx if batch_idx is not None else self.count)
        self.count += 1

    def per_slot_reduce(self) -> Any:
        return self.sum

    def cross_slot_reduce(self, per_slot_metrics) -> Any:
        return sum(per_slot_metrics)


def triangle_label_sum(updates: List) -> Any:
    out = 0
    for update_idx, (label_sum, batch_idx) in enumerate(updates):
        if batch_idx is not None:
            out += batch_idx * label_sum
        else:
            out += update_idx * label_sum
    return out


class StepableLRScheduler(torch.optim.lr_scheduler._LRScheduler):
    def get_lr(self) -> List[float]:
        return [self._step_count for _ in self.base_lrs]


def get_onevar_model(n=1) -> torch.nn.Module:
    model = torch.nn.Linear(n, n, False)
    # Manually initialize the weight(s) to 0.
    model.weight.data.fill_(0)
    return model


class MetricsCallback(pytorch.PyTorchCallback):
    def __init__(self):
        self.validation_metrics = []
        self.training_metrics = []
        self.batch_metrics = []

    def on_validation_end(self, metrics: Dict[str, Any]) -> None:
        self.validation_metrics.append(metrics)

    def on_training_workload_end(
        self, avg_metrics: Dict[str, Any], batch_metrics: Dict[str, Any]
    ) -> None:
        self.training_metrics.append(avg_metrics)
        self.batch_metrics += batch_metrics


class CheckpointCallback(pytorch.PyTorchCallback):
    def __init__(self):
        self.uuids = []

    def on_checkpoint_upload_end(self, uuid: str) -> None:
        self.uuids.append(uuid)


class BaseOneVarTrial(pytorch.PyTorchTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        # The "features" hparam is only for TestPyTorchTrial.test_restore_invalid_checkpoint
        self.model = context.wrap_model(
            get_onevar_model(n=self.context.get_hparams().get("features", 1))
        )

        self.lr = 0.001

        opt = torch.optim.SGD(self.model.parameters(), self.lr)
        self.opt = context.wrap_optimizer(opt)

        self.loss_fn = torch.nn.MSELoss()

        self.cls_reducer = context.wrap_reducer(TriangleLabelSum(), name="cls_reducer")
        self.fn_reducer = context.wrap_reducer(triangle_label_sum, name="fn_reducer")

        self.hparams = self.context.get_hparams()
        self.metrics_callback = MetricsCallback()
        self.checkpoint_callback = CheckpointCallback()
        if self.hparams.get("disable_dataset_reproducibility_checks"):
            self.context.experimental.disable_dataset_reproducibility_checks()

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, label = batch

        self.cls_reducer.update(sum(label), batch_idx)
        self.fn_reducer.update((sum(label), batch_idx))

        # Measure the weight right now.
        w_before = self.model.weight.data.item()

        # Calculate expected values for loss (eq 1) and weight (eq 4).
        loss_exp = (label[0] - data[0] * w_before) ** 2
        w_exp = w_before + 2 * self.lr * data[0] * (label[0] - (data[0] * w_before))

        output = self.model(data)
        loss = self.loss_fn(output, label)

        self.context.backward(loss)
        self.context.step_optimizer(self.opt)

        # Measure the weight after the update.
        w_after = self.model.weight.data.item()

        # Return values that we can compare as part of the tests.
        return {
            "loss": loss,
            "loss_exp": loss_exp,
            "w_before": w_before,
            "w_after": w_after,
            "w_exp": w_exp,
            "output": output,
        }

    @staticmethod
    def check_batch_metrics(
        metrics: Dict[str, Any],
        batch_idx: int,
        metric_keyname_pairs: Iterable[Tuple[str, str]],
        atol=1e-6,
    ) -> None:
        """Check that given metrics are equal or close enough to each other."""
        for k_a, k_b in metric_keyname_pairs:
            m_a, m_b = metrics[k_a], metrics[k_b]
            try:
                assert torch.isclose(
                    m_a, m_b, atol=atol
                ), f"Metrics {k_a}={m_a} and {k_b}={m_b} do not match at batch {batch_idx}"
            except TypeError:
                assert np.allclose(
                    m_a, m_b, atol=atol
                ), f"Metrics {k_a}={m_a} and {k_b}={m_b} do not match at batch {batch_idx}"

    def build_training_data_loader(self) -> torch.utils.data.DataLoader:
        if self.hparams["dataloader_type"] == "determined":
            return pytorch.DataLoader(
                OnesDataset(), batch_size=self.context.get_per_slot_batch_size()
            )
        elif self.hparams["dataloader_type"] == "torch":
            dataset = OnesDataset()
            seed = self.context.get_trial_seed()
            num_workers = self.context.distributed.get_size()
            rank = self.context.distributed.get_rank()
            batch_size = self.context.get_per_slot_batch_size()
            skip_batches = self.context.get_initial_batch()

            sampler = torch.utils.data.SequentialSampler(dataset)
            sampler = samplers.ReproducibleShuffleSampler(sampler, seed)
            sampler = samplers.RepeatSampler(sampler)
            sampler = samplers.DistributedSampler(sampler, num_workers=num_workers, rank=rank)
            batch_sampler = torch.utils.data.BatchSampler(sampler, batch_size, drop_last=False)
            batch_sampler = samplers.SkipBatchSampler(batch_sampler, skip_batches)

            return torch.utils.data.DataLoader(dataset, batch_sampler=batch_sampler)
        else:
            raise ValueError(f"unknown dataloader_type: {self.hparams['dataloader_type']}")

    def build_validation_data_loader(self) -> torch.utils.data.DataLoader:
        if self.hparams["dataloader_type"] == "determined":
            return pytorch.DataLoader(
                OnesDataset(), batch_size=self.context.get_per_slot_batch_size()
            )
        elif self.hparams["dataloader_type"] == "torch":
            dataset = OnesDataset()
            num_workers = self.context.distributed.get_size()
            rank = self.context.distributed.get_rank()
            batch_size = self.context.get_per_slot_batch_size()

            sampler = torch.utils.data.SequentialSampler(dataset)
            sampler = samplers.DistributedSampler(sampler, num_workers=num_workers, rank=rank)
            batch_sampler = torch.utils.data.BatchSampler(sampler, batch_size, drop_last=False)

            return torch.utils.data.DataLoader(dataset, batch_sampler=batch_sampler)
        else:
            raise ValueError(f"unknown dataloader_type: {self.hparams['dataloader_type']}")

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"metrics": self.metrics_callback, "checkpoint": self.checkpoint_callback}


class OneVarTrial(BaseOneVarTrial):
    _searcher_metric = "val_loss"

    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        data, label = batch

        self.cls_reducer.update(sum(label), None)
        self.fn_reducer.update((sum(label), None))

        loss = self.loss_fn(self.model(data), label)
        return {"val_loss": loss}


class OneVarTrialWithMultiValidation(OneVarTrial):
    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        data, labels = batch
        output = self.model(data)
        val_loss = self.loss_fn(output, labels)
        mse = torch.mean(torch.square(output - labels))

        return {"val_loss": val_loss, "mse": mse}


class OneVarTrialPerMetricReducers(OneVarTrialWithMultiValidation):
    def evaluation_reducer(self) -> Dict[str, pytorch.Reducer]:
        return {"val_loss": pytorch.Reducer.AVG, "mse": pytorch.Reducer.AVG}


class OneVarTrialWithTrainingMetrics(OneVarTrial):
    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        labels = cast(torch.Tensor, labels)
        loss = self.loss_fn(output, labels)
        mse = torch.mean(torch.square(output - labels))

        self.context.backward(loss)
        self.context.step_optimizer(self.opt)
        return {"loss": loss, "mse": mse}


class AMPTestDataset(OnesDataset):
    STAGE_DATUM = {
        "one": 1.0,
        "zero": 0.0,
        "small": 2e-14,
        "large": 2e4,
    }

    def __init__(self, stages: List[str], aggregation_freq: int = 1) -> None:
        self.stages = stages
        self._agg_freq = aggregation_freq

    def __len__(self) -> int:
        return len(self.stages) * self._agg_freq

    def __getitem__(self, index: int) -> Tuple:
        x = self.STAGE_DATUM[self.stages[index // self._agg_freq]]
        return torch.Tensor([float(x)]), torch.Tensor([float(x)])


class OneVarAMPBaseTrial(OneVarTrial):
    _init_scale = None
    _growth_interval = None
    _stages = (
        5 * ["one"]
        + 1 * ["large"]
        + 4 * ["one"]
        + 1 * ["small"]
        + 4 * ["one"]
        + 1 * ["zero"]
        + 4 * ["one"]
        + []
    )

    def __init__(self, context: pytorch.PyTorchTrialContext):
        super().__init__(context)
        self._agg_freq = self.context._aggregation_frequency

    def build_training_data_loader(self) -> torch.utils.data.DataLoader:
        return pytorch.DataLoader(
            AMPTestDataset(self._stages, self._agg_freq),
            batch_size=self.context.get_per_slot_batch_size(),
        )


class OneVarApexAMPTrial(OneVarAMPBaseTrial):
    _growth_interval = 2000

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)
        self.model, self.optimizer = self.context.configure_apex_amp(
            models=self.model,
            optimizers=self.opt,
            opt_level="O2",
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        scale_before = apex.amp.state_dict()["loss_scaler0"]["loss_scale"]
        metrics = super().train_batch(batch, epoch_idx, batch_idx)
        metrics["scale_before"] = scale_before
        metrics["scale"] = apex.amp.state_dict()["loss_scaler0"]["loss_scale"]
        metrics["stage"] = self._stages[batch_idx // self._agg_freq]
        return metrics


class OneVarAutoAMPTrial(OneVarAMPBaseTrial):
    _init_scale = 65536
    _growth_interval = 4

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        context.experimental.use_amp()
        # HACK: overwrite the scaler with a manually configured one, which
        #  is not something we don't actually allow with the use_amp() API.
        context._scaler = torch.cuda.amp.GradScaler(
            init_scale=self._init_scale,
            growth_interval=self._growth_interval,
        )
        super().__init__(context)
        self.scaler = self.context._scaler

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        scale_before = self.scaler.get_scale()
        metrics = super().train_batch(batch, epoch_idx, batch_idx)
        metrics["scale_before"] = scale_before
        # self.scaler.update() gets called after this method returns
        metrics["stage"] = self._stages[batch_idx // self._agg_freq]
        return metrics


class OneVarManualAMPTrial(OneVarAMPBaseTrial):
    _init_scale = 65536
    _growth_interval = 4

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.scaler = context.wrap_scaler(
            torch.cuda.amp.GradScaler(
                init_scale=self._init_scale, growth_interval=self._growth_interval
            )
        )
        super().__init__(context)

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, label = batch

        scale_before = self.scaler.get_scale()

        # Measure the weight right now.
        w_before = self.model.weight.data.item()

        # Calculate expected values for loss (eq 1) and weight (eq 4).
        loss_exp = (label[0] - data[0] * w_before) ** 2
        w_exp = w_before + 2 * self.lr * data[0] * (label[0] - (data[0] * w_before))

        with torch.cuda.amp.autocast():
            output = self.model(data)
            loss = self.loss_fn(output, label)

        scaled_loss = self.scaler.scale(loss)
        self.context.backward(scaled_loss)
        self.context.step_optimizer(self.opt, scaler=self.scaler)
        if (batch_idx + 1) % self._agg_freq == 0:
            self.scaler.update()

        # Measure the weight after the update.
        w_after = self.model.weight.data.item()

        # Return values that we can compare as part of the tests.
        return {
            "stage": self._stages[batch_idx // self._agg_freq],
            "scale_before": scale_before,
            "scale": self.scaler.get_scale(),
            "loss": loss,
            "loss_exp": loss_exp,
            "w_before": w_before,
            "w_after": w_after,
            "w_exp": w_exp,
            "output": output,
        }

    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        data, label = batch
        with torch.cuda.amp.autocast():
            output = self.model(data)
            loss = self.loss_fn(output, label)
        return {"val_loss": loss}


class OneVarApexAMPWithNoopScalerTrial(OneVarApexAMPTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.scaler = context.wrap_scaler(
            torch.cuda.amp.GradScaler(
                init_scale=self._init_scale,
                growth_interval=self._growth_interval,
                enabled=False,
            )
        )
        super().__init__(context)


class OneVarManualAMPWithNoopApexTrial(OneVarManualAMPTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)
        self.model, self.optimizer = self.context.configure_apex_amp(
            models=self.model,
            optimizers=self.opt,
            opt_level="O2",
            enabled=False,
        )


class OneVarTrialCustomEval(BaseOneVarTrial):
    _searcher_metric = "val_loss"

    def evaluate_full_dataset(self, data_loader: torch.utils.data.DataLoader) -> Dict[str, Any]:
        loss_sum = 0.0
        for data, labels in iter(data_loader):
            if torch.cuda.is_available():
                data, labels = data.cuda(), labels.cuda()
            output = self.model(data)
            loss_sum += self.loss_fn(output, labels)

        loss = loss_sum / len(data_loader)
        return {"val_loss": loss}


class OneVarTrialAccessContext(BaseOneVarTrial):
    _searcher_metric = "val_loss"

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)

        self.model_a = self.context.wrap_model(get_onevar_model())
        self.model_b = self.context.wrap_model(get_onevar_model())
        self.opt_a = self.context.wrap_optimizer(
            torch.optim.SGD(self.model_a.parameters(), self.context.get_hparam("learning_rate"))
        )
        self.opt_b = self.context.wrap_optimizer(
            torch.optim.SGD(self.model_b.parameters(), self.context.get_hparam("learning_rate"))
        )
        self.lrs_a = self.context.wrap_lr_scheduler(
            StepableLRScheduler(self.opt_a),
            step_mode=pytorch.LRScheduler.StepMode(
                self.context.get_hparam("lr_scheduler_step_mode")
            ),
        )
        self.lrs_b = self.context.wrap_lr_scheduler(
            StepableLRScheduler(self.opt_b),
            step_mode=pytorch.LRScheduler.StepMode(
                self.context.get_hparam("lr_scheduler_step_mode")
            ),
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        assert self.context.models
        assert self.context.optimizers
        assert self.context.lr_schedulers

        data, labels = batch
        output = self.model_a(data)
        loss = torch.nn.functional.binary_cross_entropy(output, labels.contiguous().view(-1, 1))

        self.context.backward(loss)
        self.context.step_optimizer(self.opt_a)

        return {"loss": loss}

    def evaluate_batch(self, batch: pytorch.TorchData, batch_idx: int) -> Dict[str, Any]:
        assert self.context.models
        assert self.context.optimizers
        assert self.context.lr_schedulers

        data, labels = batch
        output = self.model_a(data)
        loss = self.loss_fn(output, labels)

        return {"val_loss": loss}


class OneVarTrialGradClipping(OneVarTrial):
    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = self.loss_fn(output, labels)

        self.context.backward(loss)

        if "gradient_clipping_l2_norm" in self.context.get_hparams():
            self.context.step_optimizer(
                self.opt,
                clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(
                    params, self.context.get_hparam("gradient_clipping_l2_norm")
                ),
            )

        elif "gradient_clipping_value" in self.context.get_hparams():
            self.context.step_optimizer(
                self.opt,
                clip_grads=lambda params: torch.nn.utils.clip_grad_value_(
                    params, self.context.get_hparam("gradient_clipping_value")
                ),
            )

        else:
            self.context.step_optimizer(self.opt)

        return {"loss": loss}


class OneVarTrialWithNonScalarValidation(BaseOneVarTrial):
    _searcher_metric = "mse"

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)

        self.model = self.context.wrap_model(get_onevar_model())
        self.opt = self.context.wrap_optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        data, labels = batch
        output = self.model(data)
        loss = self.loss_fn(output, labels)

        self.context.backward(loss)
        self.context.step_optimizer(self.opt)
        return {"loss": loss}

    def evaluate_full_dataset(self, data_loader: torch.utils.data.DataLoader) -> Dict[str, Any]:
        predictions = []
        mse_sum = 0.0
        for data, labels in iter(data_loader):
            if torch.cuda.is_available():
                data, labels = data.cuda(), labels.cuda()
            output = self.model(data)
            predictions.append(output)
            mse_sum += torch.mean(torch.square(output - labels))

        mse = mse_sum / len(data_loader)
        return {"predictions": predictions, "mse": mse}


class OneVarTrialWithLRScheduler(OneVarTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)

        self.model = self.context.wrap_model(get_onevar_model())
        self.opt = self.context.wrap_optimizer(
            torch.optim.SGD(self.model.parameters(), self.context.get_hparam("learning_rate"))
        )

        self.lr_scheduler = self.context.wrap_lr_scheduler(
            StepableLRScheduler(self.opt),
            step_mode=pytorch.LRScheduler.StepMode(
                self.context.get_hparam("lr_scheduler_step_mode")
            ),
        )

    def train_batch(
        self, batch: pytorch.TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        metrics = super().train_batch(batch, epoch_idx, batch_idx)
        lr = self.lr_scheduler.get_last_lr()[0]
        metrics["lr"] = lr

        if (
            self.context.get_hparam("lr_scheduler_step_mode")
            == pytorch.LRScheduler.StepMode.MANUAL_STEP
        ):
            self.lr_scheduler.step()
        return metrics


class EphemeralLegacyCallbackCounter(pytorch.PyTorchCallback):
    """
    Callback with legacy signature for on_training_epoch_start
    that takes no arguments. It is ephemeral: it does not implement
    state_dict and load_state_dict.
    """

    def __init__(self) -> None:
        self.legacy_on_training_epochs_start_calls = 0

    def on_training_epoch_start(self) -> None:  # noqa # This is to test for a deprecation warning.
        logging.debug(f"calling {__name__} without arguments")
        self.legacy_on_training_epochs_start_calls += 1


class OneVarTrialCallbacks(OneVarTrial):
    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        super().__init__(context)
        self.counter = pytorch_counter_callback.Counter()
        self.legacy_counter = EphemeralLegacyCallbackCounter()

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"counter": self.counter, "legacyCounter": self.legacy_counter}


if __name__ == "__main__":
    conf = yaml.safe_load(
        """
    description: test-native-api-local-test-mode
    hyperparameters:
      global_batch_size: 32
      dataloader_type: determined
    scheduling_unit: 1
    searcher:
      name: single
      metric: val_loss
      max_length:
        batches: 1
      smaller_is_better: true
    max_restarts: 0
    """
    )
    experimental.create(OneVarTrial, conf, context_dir=".", local=True, test=True)
