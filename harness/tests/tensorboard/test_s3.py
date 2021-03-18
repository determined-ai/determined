import copy
import pathlib

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


def test_s3_build() -> None:
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, default_conf
    )
    assert isinstance(manager, tensorboard.S3TensorboardManager)


def test_s3_build_missing_param() -> None:
    conf = copy.deepcopy(default_conf)
    del conf["bucket"]

    with pytest.raises(KeyError):
        env = test_util.get_dummy_env()
        tensorboard.build(env.det_cluster_id, env.det_experiment_id, env.det_trial_id, conf)


def test_s3_lifecycle(monkeypatch: monkeypatch.MonkeyPatch) -> None:
    monkeypatch.setattr("boto3.client", s3.s3_client)
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, default_conf
    )
    assert isinstance(manager, tensorboard.S3TensorboardManager)

    manager.sync()
    expected = (
        "s3_bucket",
        "uuid-123/tensorboard/experiment/1/trial/1/events.out.tfevents.example",
    )
    assert expected in manager.client.objects


def test_s3_faulty_lifecycle(monkeypatch: monkeypatch.MonkeyPatch) -> None:
    monkeypatch.setattr("boto3.client", s3.s3_faulty_client)
    env = test_util.get_dummy_env()
    manager = tensorboard.build(
        env.det_cluster_id, env.det_experiment_id, env.det_trial_id, default_conf
    )

    with pytest.raises(exceptions.S3UploadFailedError):
        manager.sync()
