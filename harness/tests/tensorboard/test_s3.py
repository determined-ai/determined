import copy
import os
import pathlib
from typing import Optional

import pytest
from _pytest import monkeypatch
from boto3 import exceptions

from determined import tensorboard
from tests import s3
from tests.tensorboard import test_util

BASE_PATH = pathlib.Path(__file__).resolve().parent.joinpath("fixtures")

default_conf = {
    "type": "s3",
    "bucket": "s3_bucket",
    "access_key": "key",
    "secret_key": "a_secret",
    "base_path": BASE_PATH,
}


@pytest.mark.parametrize("prefix", [None, "my/test/prefix/"])
def test_s3_build(prefix: Optional[str]) -> None:
    env = test_util.get_dummy_env()
    conf = copy.deepcopy(default_conf)
    conf["prefix"] = prefix
    manager = tensorboard.build(env.det_cluster_id, env.det_experiment_id, env.det_trial_id, conf)
    assert isinstance(manager, tensorboard.S3TensorboardManager)


def test_s3_build_missing_param() -> None:
    conf = copy.deepcopy(default_conf)
    del conf["bucket"]

    with pytest.raises(KeyError):
        env = test_util.get_dummy_env()
        tensorboard.build(env.det_cluster_id, env.det_experiment_id, env.det_trial_id, conf)


@pytest.mark.parametrize("prefix", [None, "my/test/prefix/"])
@pytest.mark.parametrize("async_upload", [True, False])
def test_s3_lifecycle(
    monkeypatch: monkeypatch.MonkeyPatch, prefix: Optional[str], async_upload: bool
) -> None:
    monkeypatch.setattr("boto3.client", s3.s3_client)
    env = test_util.get_dummy_env()
    conf = copy.deepcopy(default_conf)
    conf["prefix"] = prefix

    with tensorboard.build(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, conf, async_upload=async_upload
    ) as manager:
        assert isinstance(manager, tensorboard.S3TensorboardManager)

        tfevents_path = "uuid-123/tensorboard/experiment/1/trial/1/events.out.tfevents.example"

        manager.sync()
        if prefix is not None:
            tfevents_path = os.path.join(os.path.normpath(prefix).lstrip("/"), tfevents_path)
        manager.close()
        expected = (
            "s3_bucket",
            tfevents_path,
        )
        assert expected in manager.client.objects


def test_invalid_prefix(monkeypatch: monkeypatch.MonkeyPatch) -> None:
    env = test_util.get_dummy_env()
    conf = copy.deepcopy(default_conf)
    conf["prefix"] = "my/invalid/../prefix"

    with pytest.raises(ValueError):
        tensorboard.build(env.det_cluster_id, env.det_experiment_id, env.det_trial_id, conf)


def test_s3_faulty_lifecycle(monkeypatch: monkeypatch.MonkeyPatch) -> None:
    monkeypatch.setattr("boto3.client", s3.s3_faulty_client)
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, default_conf
    )

    with pytest.raises(exceptions.S3UploadFailedError):
        manager.sync()
