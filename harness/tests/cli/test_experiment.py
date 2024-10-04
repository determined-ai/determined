import dataclasses
import os
import tempfile
from unittest import mock

import pytest
import requests_mock
from responses import matchers

from determined.cli import cli
from determined.common import api
from determined.common.api import bindings
from tests.cli import util
from tests.fixtures import api_responses


@dataclasses.dataclass
class CliArgs:
    master: str
    experiment_id: int
    user: str = "test"
    password: str = "test"
    polling_interval: float = 0.01  # Short polling interval so we can run tests quickly


@mock.patch("determined.common.api.authentication.login_with_cache")
def test_wait_returns_error_code_when_experiment_errors(
    login_with_cache_mock: mock.MagicMock,
    requests_mock: requests_mock.Mocker,
) -> None:
    master = "http://localhost:8888"
    login_with_cache_mock.return_value = api.Session(master, "test", "test", None)
    exp = api_responses.sample_get_experiment(id=1, state=bindings.experimentv1State.COMPLETED)
    args = CliArgs(master=master, experiment_id=1)
    exp.experiment.state = bindings.experimentv1State.ERROR
    requests_mock.get(
        f"/api/v1/experiments/{args.experiment_id}",
        status_code=200,
        json=exp.to_json(),
    )
    with pytest.raises(SystemExit) as e:
        cli.experiment.wait(args)  # type: ignore
    assert e.value.code == 1


def test_experiment_continue_config_file_and_cli_args() -> None:
    """
    Make sure that `det e continue` honors --config-file and --config, giving priority to --config.
    """
    with util.standard_cli_rsps() as rsps:
        rsps.post(
            "http://localhost:8080/api/v1/experiments/continue",
            status=200,
            match=[
                matchers.json_params_matcher(
                    params={
                        "id": 1,
                        "overrideConfig": "hyperparameters: {source: cli}\nname: the_new_name\n",
                    },
                    strict_match=True,
                ),
            ],
            json={
                "experiment": bindings.v1Experiment(
                    archived=False,
                    config={},
                    id=2,
                    jobId="the-job-id",
                    name="the_new_name",
                    numTrials=1,
                    originalConfig="does.not.matter.",
                    projectId=1,
                    projectOwnerId=1,
                    searcherType="single",
                    startTime="0",
                    state=bindings.experimentv1State.RUNNING,
                    username="@snoopdoggydawg",
                ).to_json(),
            },
        )

        # Don't use NamedTemporaryFile, since it would make the file inaccessible by path on
        # Windows after this.
        # (see https://docs.python.org/3/library/tempfile.html#tempfile.NamedTemporaryFile)
        try:
            fd, combined_path = tempfile.mkstemp(prefix="test-experiment-continue")
            with open(fd, "w") as f:
                f.write(
                    """
                    name: the_new_name
                    hyperparameters:
                        source: file
                """
                )
            cli.main(
                [
                    "experiment",
                    "continue",
                    "1",
                    "--config-file",
                    combined_path,
                    "--config",
                    "hyperparameters.source=cli",
                ],
            )
        finally:
            os.unlink(combined_path)
