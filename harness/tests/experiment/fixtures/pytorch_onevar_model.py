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

TODO(DET-1597): migrate the all pytorch XOR trial unit tests to variations of the OneVarTrial.
"""

from typing import Any, Dict, Iterable, List, Optional, Tuple

import numpy as np
import torch
import yaml

from determined import experimental, pytorch
from determined.pytorch import samplers

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


class OneVarTrial(pytorch.PyTorchTrial):
    _searcher_metric = "val_loss"

    def __init__(self, context: pytorch.PyTorchTrialContext) -> None:
        self.context = context

        model = torch.nn.Linear(1, 1, False)

        # Manually initialize the one weight to 0.
        model.weight.data.fill_(0)

        self.model = context.wrap_model(model)

        self.lr = 0.001

        opt = torch.optim.SGD(self.model.parameters(), self.lr)
        self.opt = context.wrap_optimizer(opt)

        self.loss_fn = torch.nn.MSELoss()

        self.cls_reducer = context.wrap_reducer(TriangleLabelSum(), name="cls_reducer")
        self.fn_reducer = context.wrap_reducer(triangle_label_sum, name="fn_reducer")

        self.hparams = self.context.get_hparams()
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

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, label = batch

        self.cls_reducer.update(sum(label), None)
        self.fn_reducer.update((sum(label), None))

        loss = self.loss_fn(self.model(data), label)
        return {"val_loss": loss}

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


class AMPTestDataset(OnesDataset):
    STAGE_DATUM = {
        "one": 1.0,
        "zero": 0.0,
        "small": 2e-14,
        "large": 2e4,
    }

    def __init__(self, stages: Iterable[str]) -> None:
        self.stages = stages

    def __len__(self) -> int:
        return len(self.stages)

    def __getitem__(self, index: int) -> Tuple:
        x = self.STAGE_DATUM[self.stages[index]]
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

    def build_training_data_loader(self) -> torch.utils.data.DataLoader:
        return pytorch.DataLoader(
            AMPTestDataset(self._stages), batch_size=self.context.get_per_slot_batch_size()
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
        metrics["stage"] = self._stages[batch_idx]
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
        metrics["stage"] = self._stages[batch_idx]
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
        self.scaler.update()

        # Measure the weight after the update.
        w_after = self.model.weight.data.item()

        # Return values that we can compare as part of the tests.
        return {
            "stage": self._stages[batch_idx],
            "scale_before": scale_before,
            "scale": self.scaler.get_scale(),
            "loss": loss,
            "loss_exp": loss_exp,
            "w_before": w_before,
            "w_after": w_after,
            "w_exp": w_exp,
            "output": output,
        }

    def evaluate_batch(self, batch: pytorch.TorchData) -> Dict[str, Any]:
        data, label = batch
        with torch.cuda.amp.autocast():
            output = self.model(data)
            loss = self.loss_fn(output, label)
        return {"val_loss": loss}


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
