import logging
import os
import pathlib
from collections import defaultdict
from typing import Dict, List, Optional, Set, Tuple

import tensorflow as tf
from tensorflow.python.training.checkpoint_state_pb2 import CheckpointState

from determined import tensorboard
from determined_common import check

# TODO: DET-175 direct checkpoint file manipulation is deprecated in TF 1.13.


class Checkpoint:
    """
    The metadata about a checkpoint.

    A checkpoint is a collection of files sharing the same prefix, called a
    basename in this code, along with entries in a checkpoint state file. The
    checkpoint state file [1] contains (possibly absolute) paths to basenames in
    the order that the checkpoints were taken.

    - state_file
        path to the state file
    - name
        the common prefix for checkpoint files ("model" by default)
    - state
        the contents of the state_file, a `CheckpointState` instance
    - paths
        a dictionary of paths for each checkpoint basename (e.g., "input.ckpt-0")

    The contents of a typical checkpoint directory:
    - checkpoint
        the "model" checkpoint state
    - model.ckpt-0.data-00000-of-00001, model.ckpt-0.index, model.ckpt-0.meta
        the "model" checkpoint data, index and metadata for step 0
    - checkpoint_input
        the "input" checkpoint state
    - graph.pbtxt
        protobuf of graph in text form; typically present but not relevant to
        `Checkpoint`
    - ...

    [1] https://github.com/tensorflow/tensorflow/blob/master/tensorflow/python/training/checkpoint_state.proto  # noqa
    """

    def __init__(
        self, state_file: str, name: str, state: CheckpointState, paths: Dict[str, List[str]]
    ):
        self.name = name
        self.state_file = state_file
        self.state = state
        self.paths = paths


def split_checkpoint_filename(filename: str) -> Tuple[str, str]:
    """
    Given a filename like "model.ckpt-20.index" return ("model",
    "model.ckpt-20") or ("", "") if the filename is not a valid checkpoint file.
    """
    parts = filename.rsplit(".", 2)
    if len(parts) < 3 or not parts[1].startswith("ckpt-"):
        return ("", "")

    cname = parts[0]
    basename = ".".join([cname, parts[1]])
    return (cname, basename)


def _scan_checkpoint_directory(checkpoint_dir: str) -> List[Checkpoint]:
    """
    Construct checkpoint metadata directly from a directory.

    State files are sometimes out of sync with directory contents. Insert
    additional orphaned checkpoint files and prune missing files. To be
    conservative, we prefer data in checkpoint states, if correct, to those
    gathered from reading the directory.
    """

    # Phase 1: Scan directory.

    # `checkpoint_state_files` is a list of (cname, full path) tuples for each
    # checkpoint state file in the directory.
    checkpoint_state_files = []
    scanned_basenames = set()
    checkpoint_paths = defaultdict(
        lambda: defaultdict(list)
    )  # type: Dict[str, Dict[str, List[str]]]
    with os.scandir(checkpoint_dir) as it:
        for f in it:
            if not f.is_file():
                continue

            if f.name.startswith("checkpoint_"):
                cname = f.name[len("checkpoint_") :]
                checkpoint_state_files.append((cname, f.path))
                continue
            elif f.name == "checkpoint":
                cname = "model"
                checkpoint_state_files.append((cname, f.path))
                continue

            cname, basename = split_checkpoint_filename(f.name)
            if not cname:
                continue

            scanned_basenames.add(basename)
            checkpoint_paths[cname][basename].append(f.path)

    # Phase 2: Read data from state files.

    checkpoints = {}
    for cname, path in checkpoint_state_files:
        latest_filename = os.path.basename(path)
        state = tf.train.get_checkpoint_state(checkpoint_dir, latest_filename=latest_filename)
        checkpoints[cname] = Checkpoint(
            state_file=path, name=cname, state=state, paths=checkpoint_paths[cname]
        )

    # Phase 3: Merge scanned data with state data, preferring state data.

    for cname, checkpoint in checkpoints.items():
        old_ts = checkpoint.state.all_model_checkpoint_timestamps
        old_paths = checkpoint.state.all_model_checkpoint_paths
        # Use 0.0 as the default timestamp if none exists previously.
        if not old_ts:
            old_ts = [0.0] * len(old_paths)

        items = [(os.path.join(checkpoint_dir, b), 0.0) for b in checkpoint_paths[cname]]
        check.check_eq(len(old_paths), len(old_ts))
        items.extend(zip(old_paths, old_ts))

        seen = set()  # type: Set[str]
        new_items = []
        for path, ts in reversed(items):
            basename = os.path.basename(path)
            if basename not in scanned_basenames:
                continue
            elif basename in seen:
                continue
            seen.add(basename)
            new_items.append((path, ts))

        if not new_items:
            raise Exception(
                "No checkpoint files found for {} checkpoint in directory {}".format(
                    cname, checkpoint_dir
                )
            )

        new_paths, new_ts = zip(*reversed(new_items))

        all_model_checkpoint_timestamps = None
        last_preserved_timestamp = None
        if checkpoint.state.all_model_checkpoint_timestamps is not None:
            all_model_checkpoint_timestamps = new_ts
            last_preserved_timestamp = new_ts[-1]

        check.check_eq(
            new_paths[-1],
            checkpoint.state.model_checkpoint_path,
            "Most recent checkpoint path should not change",
        )
        checkpoint.state = tf.compat.v1.train.generate_checkpoint_state_proto(
            checkpoint_dir,
            new_paths[-1],
            all_model_checkpoint_paths=new_paths,
            all_model_checkpoint_timestamps=all_model_checkpoint_timestamps,
            last_preserved_timestamp=last_preserved_timestamp,
        )

    return list(checkpoints.values())


def move_tf_events(event_dir: pathlib.Path) -> None:
    """
    Given a TensorFlow Estimator model directory, find all nested tfevents
    files and move them to the TensorBoard log directory. For the most part, we
    expect only one tfevents file in the root_dir tree. This recursive search
    for tfevents is an extra measure to make sure we do not miss any events.
    """
    tensorboard_dir = tensorboard.get_base_path({})
    for event_file in event_dir.rglob("*tfevents*"):
        event_file.rename(tensorboard_dir.joinpath(event_file.name))


def _cleanup_after_train_step(model_dir: pathlib.Path) -> None:
    # TF event files are written out during training by estimators. We move
    # them to the tensorboard directory so that they can be saved to persistent
    # storage.
    move_tf_events(model_dir)

    # By default the Estimator API is configured to accumulate checkpoints
    # in the model directory after every train() invocation. To avoid
    # wasting disk space, we delete all but the most recent checkpoint at
    # the end of each training step. A checkpoint is always computed at the
    # end of the train() invocation, so this checkpoint is guaranteed to be
    # the most recent state of the model.
    delete_all_checkpoints_except_most_recent(str(model_dir))


def _cleanup_after_validation_step(model_dir: pathlib.Path, is_chief: bool) -> None:
    if is_chief:
        move_tf_events(model_dir)


def delete_all_checkpoints_except_most_recent(model_dir: str) -> None:
    """
    Given a TensorFlow Estimator model directory, delete all of the checkpoints
    except the most recent and update the checkpoint state.
    """
    for checkpoint in _scan_checkpoint_directory(model_dir):
        all_paths = checkpoint.state.all_model_checkpoint_paths
        for path in all_paths[:-1]:
            basename = os.path.basename(path)
            for p in checkpoint.paths[basename]:
                logging.debug("Deleting non-recent checkpoint file %s", p)
                os.remove(p)

        tf.compat.v1.train.update_checkpoint_state(
            model_dir,
            model_checkpoint_path=all_paths[-1],
            all_model_checkpoint_paths=[all_paths[-1]],
            latest_filename=os.path.basename(checkpoint.state_file),
        )


def load_global_step_from_checkpoint(checkpoint_dir: str) -> Optional[tf.Tensor]:
    checkpoint = tf.train.latest_checkpoint(checkpoint_dir)
    if checkpoint is None:
        return None

    reader = tf.compat.v1.train.NewCheckpointReader(checkpoint)
    return reader.get_tensor(tf.compat.v1.GraphKeys.GLOBAL_STEP)


def _update_checkpoint_path_in_state_file(model_dir: pathlib.Path) -> None:
    """
    In checkpoint state files, the paths to checkpoint files can be absolute.
    This function updates the directory portion of paths in a state file to be
    `model_dir` instead. This is useful when copying checkpoints between
    machines.
    """
    for checkpoint in _scan_checkpoint_directory(str(model_dir)):
        new_paths = [
            str(model_dir.joinpath(os.path.basename(v)))
            for v in checkpoint.state.all_model_checkpoint_paths
        ]
        tf.compat.v1.train.update_checkpoint_state(
            str(model_dir),
            new_paths[-1],
            all_model_checkpoint_paths=new_paths,
            latest_filename=os.path.basename(checkpoint.state_file),
            all_model_checkpoint_timestamps=checkpoint.state.all_model_checkpoint_timestamps,
            last_preserved_timestamp=checkpoint.state.last_preserved_timestamp,
        )
