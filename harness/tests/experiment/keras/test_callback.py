import contextlib
import json
import os
import pathlib
import re
import subprocess
import sys
from typing import Any, Callable, Dict, Iterator, Optional, Tuple, Union
from unittest import mock

import keras
import numpy as np
import pytest
import tensorflow as tf

import determined as det
import determined.keras
from determined import core
from determined.common import storage
from tests.experiment import utils


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
    class StorageManagerForTesting(storage.SharedFSStorageManager):
        @contextlib.contextmanager
        def restore_path(
            self, src: str, selector: Optional[storage.Selector] = None
        ) -> Iterator[pathlib.Path]:
            events.append(("restore_path:enter", None))
            try:
                with super().restore_path(src, selector) as x:
                    yield x
            finally:
                events.append(("restore_path:exit", None))

    storage_manager = StorageManagerForTesting(path)
    checkpoint = core.DummyCheckpointContext(distributed, storage_manager)

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


class DeterminedCallbackForTesting(det.keras.DeterminedCallback):
    """
    For testing purposes, log events that happen during training for evaluation after training.
    """

    def __init__(self, events: utils.Events, *args: Any, **kwargs: Any) -> None:
        self.events = events
        self.first_train_batch_end = False
        super().__init__(*args, **kwargs)

    def on_train_begin(self, logs: Any) -> None:
        super().on_train_begin(logs)
        weight = self.model.layers[0].get_weights()[0][0]
        fourdigits = "%.4f" % weight
        self.events.append((f"after_train_begin:{fourdigits}", weight))

    def on_train_batch_end(self, batch: int, logs: Any) -> None:
        if not self.first_train_batch_end:
            self.first_train_batch_end = True
            self.events.append(("first_train_batch_end", None))
        super().on_train_batch_end(batch, logs)

    def on_epoch_end(self, epoch: int, logs: Any) -> None:
        self.events.append((f"before_epoch_end:{epoch}", logs))
        super().on_epoch_end(epoch, logs)
        self.events.append((f"after_epoch_end:{epoch}", logs))

    def on_train_end(self, logs: Any) -> None:
        self.events.append(("before_train_end", None))
        super().on_train_end(logs)

    def save_model(
        self, model: keras.models.Model, path: str, distributed: core.DistributedContext
    ) -> None:
        super().save_model(model, path, distributed)
        ckpt_uuid = os.path.basename(os.path.dirname(path))
        weight = self.model.layers[0].get_weights()[0][0]
        self.events.append(("save_model", (ckpt_uuid, weight)))

    def load_model(self, *args: Any, **kwargs: Any) -> None:
        super().load_model(*args, **kwargs)
        self.events.append(("load_model", None))


def build_model(eager: bool = False) -> keras.models.Model:
    layer = keras.layers.Dense(
        1, activation=None, use_bias=False, kernel_initializer="zeros", input_shape=(8,)
    )
    model = keras.models.Sequential([layer])
    model.compile(
        loss=keras.losses.MeanSquaredError(),
        optimizer=keras.optimizers.SGD(),
        run_eagerly=eager,
    )
    return model


def do_fit(
    # Basic test configuration.
    path: Union[str, pathlib.Path],
    model: Optional[keras.models.Model] = None,
    distributed: Optional[core.DistributedContext] = None,
    # DeterminedCallback settings.
    checkpoint: Optional[str] = None,
    continue_id: int = 1,
    checkpoint_epochs: int = 1,
    train_metrics_report_period: Union[str, int] = "epoch",
    # Model.compile settings.
    eager: bool = False,
    # Model.fit settings.
    epochs: int = 2,
    verbose: int = 0,
    set_preempt_on_event: Optional[str] = None,
) -> utils.Events:
    x = np.ones((64, 8))
    y = np.ones((64, 8))
    validation_data = (np.ones((64, 8)), np.ones((64, 8)))

    model = model or build_model(eager=eager)
    events = utils.Events()
    core_context, set_preempt = mock_core_context(str(path), events, distributed)

    if set_preempt_on_event:
        # Configure a hook for our Events that calls set_preempt() when a matching event arrives.
        p = re.compile(set_preempt_on_event)

        def hook(summary: str, data: Any) -> None:
            if p.search(summary):
                set_preempt()

        events.hook = hook

    det_cb = DeterminedCallbackForTesting(
        events,
        core_context,
        checkpoint=checkpoint,
        continue_id=continue_id,
        train_metrics_report_period=train_metrics_report_period,
        checkpoint_epochs=checkpoint_epochs,
    )

    model.fit(
        x=x,
        y=y,
        validation_data=validation_data,
        batch_size=8,
        epochs=epochs,
        callbacks=[det_cb],
        verbose=verbose,
    )
    return events


def check_keras_metrics(metrics: Dict[str, Any]) -> None:
    # Make sure we are filtering out size and batch, which are pointless to our UI.
    assert "size" not in metrics and "batch" not in metrics, metrics
    # Make sure we are always injecting epochs and batches.
    assert "batches" in metrics and "epochs" in metrics, metrics
    # Never allow 'val_' prefix in log names:
    # - Validation metrics come in on_test_end, and don't include 'val_' prefix.
    # - Training metrics from on_epoch_end have val_* values, which we filter out.
    # - Training metrics from on_test_batch_end do not have val_* metrics.
    # Training metrics must not contain validation metrics.
    assert not any(m.startswith("val_") for m in metrics), metrics


@pytest.mark.tensorflow
def test_basic_logic(tmp_path: pathlib.Path) -> None:
    # make sure verbose=1 doesn't puke (though we don't really check the output)
    events = do_fit(tmp_path, verbose=1)

    # Checks that:
    #   - set_status() gets called
    #   - report_metrics() gets called
    #   - report_progress() gets called
    data = utils.assert_events_match(
        events,
        "!load_model",
        "after_train_begin",
        "set_status:training",
        "set_status:validating",
        ("report_metrics:validation", "validation_metrics_sample"),
        "before_epoch_end:0",
        ("report_metrics:training", "training_metrics_sample"),
        "report_progress:0.5000",
        "set_status:checkpointing",
        "save_model",
        "after_epoch_end:0",
        "before_epoch_end:1",
        "report_progress:1.000",
        "save_model",
        "after_epoch_end:1",
        "before_train_end",
        "!save_model",  # No final checkpoint.
        "set_status:finishing",
    )
    # Check examples of training and validation metrics.
    check_keras_metrics(data["training_metrics_sample"])
    check_keras_metrics(data["validation_metrics_sample"])


# Pick this test to run eagerly because it both saves and loads checkpoints, which feel like it
# could matter if run_eagerly was set or not.
@pytest.mark.parametrize("eager", [False, True])
@pytest.mark.tensorflow
def test_save_restore_and_warm_start(tmp_path: pathlib.Path, eager: bool) -> None:
    # Train-from-scratch, then check that:
    # - initial weight is 0 (no checkpoint was loaded)
    # - initial epoch is 0 (no training state was loaded)
    # - checkpoint gets saved
    events = do_fit(tmp_path, eager=eager, checkpoint=None, continue_id=1)
    data = utils.assert_events_match(
        events,
        "!load_model",
        "after_train_begin:0.0000",
        "before_epoch_end:0",
        ("save_model", "ckpt"),
        "after_epoch_end:0",
        "before_epoch_end:1",
        "save_model",
        "after_epoch_end:1",
    )

    # Grab the checkpoint uuid and the weight from the "save_model" match.
    ckpt, weight = data["ckpt"]

    # Continue training (continue_id does match), then check that:
    # - initial weight is nonzero (checkpoint was loaded)
    # - initial epoch is nonzero (training state was loaded)
    # - steps_completed was properly restored
    # - checkpoint is not destoyed until first batch is completed
    events = do_fit(tmp_path, eager=eager, checkpoint=ckpt, continue_id=1)
    utils.assert_events_match(
        events,
        "set_status:restoring",
        "load_model",
        "after_train_begin:%.4f" % weight,
        "first_train_batch_end",
        "restore_path:exit",
        "!after_epoch_end:0",
        "before_epoch_end:1",
        "report_metrics:training:16",
        "after_epoch_end:1",
        "!after_epoch_end",  # Don't do two epochs if we started with one already from one.
    )

    # Warm-start training (continue_id does not match), then check that:
    # - initial weight is nonzero (no checkpoint was loaded)
    # - initial epoch is zero (no training state was loaded)
    # - steps_completed was properly reset
    # - checkpoint is not destoyed until first batch is completed
    events = do_fit(tmp_path, eager=eager, checkpoint=ckpt, continue_id=2)
    utils.assert_events_match(
        events,
        "set_status:restoring",
        "load_model",
        "after_train_begin:%.4f" % weight,
        "first_train_batch_end",
        "restore_path:exit",
        "report_metrics:training:8",
        "after_epoch_end:0",
        "after_epoch_end:1",
        "!after_epoch_end",
    )


@pytest.mark.tensorflow
def test_checkpoint_epochs(tmp_path: pathlib.Path) -> None:
    # Never checkpoint, except on preemption or completion
    events = do_fit(tmp_path, checkpoint_epochs=0, epochs=4)
    utils.assert_events_match(
        events,
        # The only save is after the final on_epoch_end
        "!save_model",
        "after_epoch_end:3",
        "!after_epoch_end",
        "before_train_end",
        "save_model",
    )

    # Same thing, but trigger a checkpoint mid-training.
    events = do_fit(tmp_path, checkpoint_epochs=0, set_preempt_on_event="report_progress:0.5000")
    utils.assert_events_match(
        events,
        "!save_model",  # The preemption-caused checkpoint is in on_train_end, not on_epoch_end.
        "after_epoch_end:0",
        "!after_epoch_end",
        "before_train_end",
        "save_model",
    )

    # Checkpoint every other epoch, exiting on a natural checkpoint.
    events = do_fit(tmp_path, checkpoint_epochs=2, epochs=4)
    utils.assert_events_match(
        events,
        "!save_model",
        "before_epoch_end:1",
        "save_model",
        "after_epoch_end:1",
        "!save_model",
        "before_epoch_end:3",
        "save_model",
        "after_epoch_end:3",
        # There is nothing to save in the on_train_end hook.
        "!after_epoch_end",
        "!save_model",
    )

    # Checkpoint every other epoch, and also at the end, if there is uncheckpointed work.
    events = do_fit(tmp_path, checkpoint_epochs=2, epochs=3)
    utils.assert_events_match(
        events,
        "!save_model",
        "before_epoch_end:1",
        "save_model",
        "after_epoch_end:1",
        "!save_model",
        "after_epoch_end:2",
        "!save_model",
        "!after_epoch_end",
        # Expect an on_train_end checkpoint.
        "before_train_end",
        "save_model",
    )

    # Checkpoint every other epoch, preempting after a natural checkpoint.
    events = do_fit(
        tmp_path, checkpoint_epochs=2, epochs=4, set_preempt_on_event="report_progress:0.5000"
    )
    utils.assert_events_match(
        events,
        "!save_model",
        "before_epoch_end:1",
        "save_model",
        "after_epoch_end:1",
        # No on_train_end checkpoint.
        "!after_epoch_end",
        "!save_model",
    )

    # Checkpoint every other epoch, preempting when there wasn't a checkpoint.
    events = do_fit(
        tmp_path, checkpoint_epochs=2, epochs=4, set_preempt_on_event="report_progress:0.2500"
    )
    utils.assert_events_match(
        events,
        "!save_model",
        "after_epoch_end:0",
        "!after_epcoh_end",
        "!save_model",
        # Expect an on_train_end checkpoint.
        "before_train_end",
        "save_model",
    )


@pytest.mark.tensorflow
def test_report_period(tmp_path: pathlib.Path) -> None:
    events = do_fit(tmp_path, train_metrics_report_period=3)
    # There are 8 batches per epoch.
    data = utils.assert_events_match(
        events,
        "!report_metrics:training:1",
        "!report_metrics:training:2",
        ("report_metrics:training:3", "training_metrics_sample"),
        "!report_metrics:training:4",
        "!report_metrics:training:5",
        "report_metrics:training:6",
        "!report_metrics:training:7",
        "!report_metrics:training:8",
        "report_metrics:validation:8",
        "report_metrics:training:9",
        "!report_metrics:training:10",
        "!report_metrics:training:11",
        "report_metrics:training:12",
        "!report_metrics:training:13",
        "!report_metrics:training:14",
        "report_metrics:training:15",
        "!report_metrics:training:16",
        "report_metrics:validation:16",
        "!report_metrics:training",
    )
    # Check training metrics from the non-epoch reporting codepath.
    check_keras_metrics(data["training_metrics_sample"])


# Pick this test to run eagerly because multi-gpu training, feel like it might be eager-senstive.
@pytest.mark.parametrize("eager", [False, True])
@pytest.mark.parametrize("multinode", [False, True])
@pytest.mark.skipif(len(tf.config.list_physical_devices("GPU")) < 2, reason="not enough gpus")
@pytest.mark.gpu_parallel
def test_multi_gpu(tmp_path: pathlib.Path, eager: bool, multinode: bool) -> None:
    """
    Getting an environment where this test can actually pass can be a real pain.

    If you are running on bare metal or a vm with multiple gpus, you can run the test directly,
    but you must have your nvidia driver and cuda library installations all squared away.  That is
    surprisingly difficult to achieve, at least I (rb) couldn't make it work.

    The tedious alternative, but which I find more reliable, is to run it in a docker container
    that is compatible with your GPU and driver.  I selected an NGC tensorflow image from the NGC
    image support matrix:

        https://docs.nvidia.com/deeplearning/frameworks/support-matrix/index.html

    Then I built a little dockerfile like this:

        FROM ncr.io/nvidia/tensorflow:$YOUR_NGC_IMAGE
        RUN pip install determined pytest && pip uninstall --yes determined
        ENV PYTHONUNBUFFERED=1

    Then I configured /etc/docker/daemon.json with some settings commonly used to make dtrain happy:

        {
            "runtimes": {
                "nvidia": {
                    "args": [],
                    "path": "nvidia-container-runtime"
                }
            },
            "default-shm-size": "4G",
            "default-ulimits": {
                "memlock": {
                    "Name": "memlock",
                    "Hard": -1,
                    "Soft": -1
                },
                "stack": {
                    "Name": "stack",
                    "Hard": 67108864,
                    "Soft": 67108864
                }
            }
        }

    Restarted docker:

        sudo systemctl restart docker

    Then I mounted the entire determined project into a container with that new image:

        cd /path/to/determined
        docker run -it --rm -v $PWD:$PWD --gpus=all $YOUR_CUSTOM_IMAGE

    And finally, inside the container, I navigate to the harness directory and install determined
    with the editable setting:

        cd /path/to/determined/harness
        pip install -e .

    And voila, I can finally run the tests:

        pytest -v -s --tb=native tests/experiment/keras/test_callback.py -k test_multi_gpu

    I can also edit the tests from outside the container and rerun them immediately within the
    container because I mounted the whole project into the container and used `-e` with the pip
    install.
    """

    script = os.path.join(os.path.dirname(__file__), "train.py")
    cmd = [sys.executable, script, str(tmp_path)]
    if eager:
        cmd += ["--eager"]
    # NCCL can hit failures in this test, so make it easy to debug.
    env = {**os.environ, "NCCL_DEBUG": "info"}
    if multinode:
        tf_config = {
            "cluster": {"worker": ["localhost:12345", "localhost:12346"]},
            "task": {"type": "worker", "index": 0},
        }
        # Start worker 0.
        env["TF_CONFIG"] = json.dumps(tf_config)
        env["CUDA_VISIBLE_DEVICES"] = "0"
        p1 = subprocess.Popen(cmd, env=env)
        # Start worker 1.
        tf_config["task"]["index"] = 1  # type: ignore
        env["TF_CONFIG"] = json.dumps(tf_config)
        env["CUDA_VISIBLE_DEVICES"] = "1"
        p2 = subprocess.Popen(cmd, env=env)
        ret1 = p1.wait()
        ret2 = p2.wait()
        assert ret1 == ret2 == 0, (ret1, ret2)
    else:
        env.pop("TF_CONFIG", None)
        env.pop("CUDA_VISIBLE_DEVICES", None)
        subprocess.run(cmd, check=True)


@pytest.mark.tensorflow
def test_iris() -> None:
    """
    Make sure the DeterminedCallback-based iris example works.
    """
    cmd = [sys.executable, utils.cv_examples_path("iris_tf_keras/train.py"), "--epochs", "1"]
    subprocess.run(cmd, check=True)
