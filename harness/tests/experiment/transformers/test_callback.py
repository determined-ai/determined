import pathlib
import re
from typing import Any, Callable, Dict, Optional, Tuple
from unittest import mock

import numpy as np
import torch
import transformers

import determined as det
import determined.transformers
from determined import core
from determined.common import storage
from tests.experiment import utils
from tests.launch import test_util


def mock_core_context(
    path: str, events: utils.Events, distributed: Optional[core.DistributedContext] = None
) -> Tuple[core.Context, Callable[[], None]]:
    """
    Returns a core_context and a set_preempt() callable.

    The core_context is partially mocked to support triggering preemption from test code and to log
    all reports to the provided Events object.
    """
    # Set up a functional DistributedContext.
    distributed = distributed or core.DummyDistributedContext()
    # Set up a functional CheckpointContext.
    storage_manager = storage.SharedFSStorageManager(path)

    class DummyCheckpointContext(core.DummyCheckpointContext):
        def _report_checkpoint(
            self,
            storage_id: str,
            resources: Optional[Dict[str, int]] = None,
            metadata: Optional[Dict[str, Any]] = None,
        ) -> None:
            events.append(("report_checkpoint", storage_id))
            super()._report_checkpoint(storage_id, resources, metadata)

    checkpoint = DummyCheckpointContext(distributed, storage_manager)

    # Mock everything else, logging report-like calls to events.

    def report_metrics(group: str, steps_completed: int, metrics: Any) -> None:
        events.append((f"report_metrics:{group}:{steps_completed}", metrics))

    def report_progress(progress: float) -> None:
        fourdigits = "%.4f" % progress
        events.append((f"report_progress:{fourdigits}", progress))

    def set_status(status: str) -> None:
        events.append((f"set_status:{status}", None))

    preempted = False

    def should_preempt() -> bool:
        nonlocal preempted
        return preempted

    core_context = mock.Mock()
    core_context.distributed = distributed
    core_context.preempt.should_preempt.side_effect = should_preempt
    core_context.checkpoint = checkpoint
    core_context.train.report_metrics.side_effect = report_metrics
    core_context.train.report_progress.side_effect = report_progress
    core_context.train.set_status.side_effect = set_status

    def set_preempt() -> None:
        nonlocal preempted
        preempted = True

    return core_context, set_preempt


class MyOneVarModel(torch.nn.Linear):  # type: ignore
    """
    Subclass torch.nn.Linear with custom behaviors to be Transformers.Trainer-friendly.
    """

    def __init__(self) -> None:
        super().__init__(1, 1, False)
        self.weight.data.fill_(0)
        self._loss_fn = torch.nn.MSELoss()

    # Signature must match key in dataset's output.
    def forward(self, x: torch.Tensor, label_y: torch.Tensor) -> Dict[str, torch.Tensor]:
        y = super().forward(x)
        loss = self._loss_fn(y, label_y)
        # We must return a dict with "loss" as a key.
        # (technically a tuple with loss as the first element is also ok)
        return {"loss": loss, "pred_y": y}


class OnesDataset(torch.utils.data.Dataset):
    def __init__(self, dataset_len: int) -> None:
        self.dataset_len = dataset_len

    def __len__(self) -> int:
        return self.dataset_len

    def __getitem__(self, index: int) -> Dict[str, torch.Tensor]:
        # Key name must match model's .forward() signature.
        return {"x": torch.Tensor([float(1)]), "label_y": torch.Tensor([float(1)])}


def compute_metrics(pred: transformers.EvalPrediction) -> Dict[str, float]:
    # Return a mean absolute error as a metric.
    return {"mae": np.abs(pred.predictions - pred.label_ids).mean()}


class DetCallbackForTesting(det.transformers.DetCallback):
    def __init__(self, events: utils.Events, *args: Any, **kwargs: Any) -> None:
        self.events = events
        super().__init__(*args, **kwargs)

    def on_train_begin(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        epoch = "%.4f" % state.epoch
        self.events.append((f"on_train_begin:{state.global_step}:{epoch}", None))

    def on_epoch_begin(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        epoch = "%.4f" % state.epoch
        self.events.append((f"on_epoch_begin:{state.global_step}:{epoch}", None))

    def on_epoch_end(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        epoch = "%.4f" % state.epoch
        weight = kwargs["model"].weight.data.item()
        self.events.append((f"before_epoch_end:{state.global_step}:{epoch}", weight))
        super().on_epoch_end(args, state, control)
        self.events.append((f"after_epoch_end:{state.global_step}:{epoch}", weight))

    def on_save(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        epoch = "%.4f" % state.epoch
        self.events.append((f"before_save:{state.global_step}:{epoch}", None))
        super().on_save(args, state, control)
        self.events.append((f"after_save:{state.global_step}:{epoch}", None))

    def on_evaluate(
        self,
        args: transformers.TrainingArguments,
        state: transformers.TrainerState,
        control: transformers.TrainerControl,
        **kwargs: Any,
    ) -> None:
        epoch = "%.4f" % state.epoch
        self.events.append((f"on_evaluate:{state.global_step}:{epoch}", None))

    def on_train_end(self, *args: Any, **kwargs: Any) -> None:
        self.events.append(("on_train_end", None))


def do_train(
    tmp_path: pathlib.Path,
    force_final_save: Optional[bool] = None,
    force_final_evaluate: Optional[bool] = None,
    set_preempt_on_event: Optional[str] = None,
    latest_checkpoint: Optional[str] = None,
    **kwargs: Any,
) -> utils.Events:
    args = transformers.TrainingArguments(
        output_dir=str(tmp_path / "trainer"), disable_tqdm=True, **kwargs
    )

    with test_util.set_mock_cluster_info(["0.0.0.0"], 0, 1) as info:
        info.trial._config = {"searcher": {"name": "single", "metric": "eval_mae"}}
        info._latest_checkpoint = latest_checkpoint

        model = MyOneVarModel()
        train_dataset = OnesDataset(64)
        eval_dataset = OnesDataset(64)

        events = utils.Events()
        core_context, set_preempt = mock_core_context(str(tmp_path / "ckpt"), events)

        if set_preempt_on_event:
            # Configure a hook for Events that calls set_preempt() when a matching event arrives.
            p = re.compile(set_preempt_on_event)

            def hook(summary: str, data: Any) -> None:
                if p.search(summary):
                    set_preempt()

            events.hook = hook

        det_cb = DetCallbackForTesting(events, core_context, args)
        if force_final_save is not None:
            det_cb._force_final_save = force_final_save
        if force_final_evaluate is not None:
            det_cb._force_final_evaluate = force_final_evaluate

        t = transformers.Trainer(
            model=model,
            args=args,
            train_dataset=train_dataset,
            eval_dataset=eval_dataset,
            compute_metrics=compute_metrics,
            callbacks=[det_cb],
        )
        # The call to train must specify the checkpoint.  We do set args.resume_from_checkpoint in
        # our DetCallback but it isn't automatically respected.
        t.train(resume_from_checkpoint=args.resume_from_checkpoint)

        return events


def check_hf_metrics(metrics: Dict[str, Any]) -> None:
    # We remove the default rounded 'epoch' metric, and the
    assert "epoch" not in metrics, metrics
    # We remove the speed metrics.
    speed_suffixes = ["_runtime", "_per_second", "_compilation_time"]
    assert not any(any(m.endswith(s) for s in speed_suffixes) for m in metrics), metrics
    # We inject "epochs" and "batches"
    assert "epochs" in metrics, metrics
    assert "batches" in metrics, metrics


def test_train_metrics(tmp_path: pathlib.Path) -> None:
    # Make sure that training metrics happen every 5 steps, as specified.
    events = do_train(
        tmp_path,
        num_train_epochs=2,
        evaluation_strategy="epoch",
        logging_steps=5,
    )
    data = utils.assert_events_match(
        events,
        "!report_metrics:training",
        ("report_metrics:training:5", "metrics"),
        "!report_metrics:training",
        "report_metrics:training:10",
        "!report_metrics:training",
        "report_metrics:training:15",
        # Trainer always logs training metrics before exiting.
        "report_metrics:training:16",
        "!report_metrics:training",
    )
    # Check non-epoch metrics.
    check_hf_metrics(data["metrics"])

    # If logging_steps aligns with our exit batch (logging_steps == len(data)), we only log once.
    events = do_train(
        tmp_path,
        num_train_epochs=1,
        evaluation_strategy="epoch",
        logging_steps=8,
    )
    data = utils.assert_events_match(
        events,
        "!report_metrics:training",
        ("report_metrics:training:8", "metrics"),
        "!report_metrics:training",
    )
    # Check epoch metrics.
    check_hf_metrics(data["metrics"])


def test_save_at_end(tmp_path: pathlib.Path) -> None:
    # We force a save even if Transformers wouldn't.
    events = do_train(
        tmp_path,
        num_train_epochs=1,
    )
    utils.assert_events_match(
        events,
        "!report_checkpoint",
        "before_save:8",
        "report_checkpoint",
        "after_save:8",
        "!report_checkpoint",
    )

    # We can override it.  Also, this tests that the previous case was valid, because it proves that
    # the save that occured was the one we forced.
    events = do_train(
        tmp_path,
        force_final_save=False,
        num_train_epochs=1,
    )
    utils.assert_events_match(
        events,
        "!report_checkpoint",
    )

    # Also, if the trainer naturally saves at that time, we don't duplicate the save.
    events = do_train(
        tmp_path,
        # force_final_save=False,
        num_train_epochs=1,
        save_steps=8,
    )
    utils.assert_events_match(
        events,
        "!report_checkpoint",
        "before_save:8",
        "report_checkpoint",
        "after_save:8",
        "!report_checkpoint",
    )

    # Same thing, but force_final_save=False to guarantee that the above test is valid (i.e. the
    # save originated with Transformers).
    events = do_train(
        tmp_path,
        force_final_save=False,
        num_train_epochs=1,
        save_steps=8,
    )
    utils.assert_events_match(
        events,
        "!report_checkpoint",
        "before_save:8",
        "report_checkpoint",
        "after_save:8",
        "!report_checkpoint",
    )

    # Save a final checkpoint if we are preempted.
    events = do_train(
        tmp_path,
        set_preempt_on_event="report_metrics:training:3",
        logging_steps=1,
        num_train_epochs=1,
    )
    utils.assert_events_match(
        events,
        "!report_checkpoint",
        "before_save:3",
        "report_checkpoint",
        "after_save:3",
        "!report_checkpoint",
    )


def test_eval(tmp_path: pathlib.Path) -> None:
    # Eval on epoch boundaries.
    # (This test also ensures we don't double-evaluate with our evaluate-at-end logic).
    events = do_train(
        tmp_path,
        num_train_epochs=2,
        evaluation_strategy="epoch",
        logging_steps=5,
    )
    data = utils.assert_events_match(
        events,
        "!report_metrics:validation",
        "!on_evaluate",
        ("report_metrics:validation:8", "metrics"),
        "on_evaluate:8",
        "!report_metrics:validation",
        "!on_evaluate",
        "report_metrics:validation:16",
        "on_evaluate:16",
        "!report_metrics:validation",
        "!on_evaluate",
    )
    # Check epoch metrics.
    check_hf_metrics(data["metrics"])

    # Eval off epoch boundaries, and once at the end.
    events = do_train(
        tmp_path,
        num_train_epochs=1,
        evaluation_strategy="steps",
        eval_steps=5,
    )
    data = utils.assert_events_match(
        events,
        "!report_metrics:validation",
        "!on_evaluate",
        ("report_metrics:validation:5", "off-epoch-metrics"),
        "on_evaluate:5",
        "!report_metrics:validation",
        "!on_evaluate",
        ("report_metrics:validation:8", "final-metrics"),
        "on_evaluate:8",
        "!report_metrics:validation",
        "!on_evaluate",
    )
    # Check non-epoch metrics, and the at-end metrics.
    check_hf_metrics(data["off-epoch-metrics"])
    check_hf_metrics(data["final-metrics"])

    # Same thing, but we can disable the evaluate-at-end.  Also this proves that our evaluate-at-end
    # was working in the previous case.
    events = do_train(
        tmp_path,
        force_final_evaluate=False,
        num_train_epochs=1,
        evaluation_strategy="steps",
        eval_steps=5,
    )
    utils.assert_events_match(
        events,
        "!report_metrics:validation",
        "!on_evaluate",
        "report_metrics:validation:5",
        "on_evaluate:5",
        "!report_metrics:validation",
        "!on_evaluate",
    )

    # Same thing, but we can disable the evaluate-at-end.  Also this proves that our evaluate-at-end
    # was working in the previous case.
    events = do_train(
        tmp_path,
        force_final_evaluate=False,
        num_train_epochs=1,
        evaluation_strategy="steps",
        eval_steps=5,
    )
    utils.assert_events_match(
        events,
        "!report_metrics:validation",
        "!on_evaluate",
        "report_metrics:validation:5",
        "on_evaluate:5",
        "!report_metrics:validation",
        "!on_evaluate",
    )

    # Never evaluate-at-end if we got preempted.
    events = do_train(
        tmp_path,
        set_preempt_on_event="report_metrics:training:3",
        num_train_epochs=1,
        logging_steps=1,
        evaluation_strategy="steps",
        eval_steps=5,
    )
    utils.assert_events_match(
        events,
        "!report_metrics:validation",
        "!on_evaluate",
    )


def test_save_and_restore(tmp_path: pathlib.Path) -> None:
    events = do_train(
        tmp_path,
        set_preempt_on_event="report_metrics:training:3",
        max_steps=5,
        logging_steps=1,
    )
    data = utils.assert_events_match(
        events,
        ("after_epoch_end", "weight"),
        ("report_checkpoint", "ckpt"),
    )

    # Make sure our next training continues from here.
    ckpt = data["ckpt"]
    ckpt_weight = data["weight"]

    # Note that model is loaded _after_ on_epoch_begin, so to know that we loaded a model we'll
    # compare weight after training one batch to the checkpoint weight (which had more than one
    # batch of training behind it).
    events = do_train(
        tmp_path,
        latest_checkpoint=ckpt,
        max_steps=1,
    )
    data = utils.assert_events_match(
        events,
        # training should continue from global_step=3
        "on_train_begin:3",
        ("after_epoch_end", "weight"),
    )

    # Model weight will be slowly moving from 0 to 1 throughout training.
    assert data["weight"] > ckpt_weight
