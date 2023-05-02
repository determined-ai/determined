"""Test functionality in the experiment module of the SDK."""
import pytest

from determined.common import api
from determined.common.api import errors
from determined.common.experimental import experiment
from tests.common import api_server


def test_experiment_ref_createable_on_extant_id() -> None:
    real_experiment_id = 2  # This experiment ID exists in the test api_server
    api_server_session = api.Session(
        master=f"{api_server.DEFAULT_HOST}:{api_server.DEFAULT_PORT}",
        user="user1",
        auth=None,
        cert=None,
    )
    with api_server.run_api_server(credentials=("user1", "password1", "token1"), ssl_keys=None):
        experiment.ExperimentReference(real_experiment_id, api_server_session)


def test_experiment_ref_not_createable_on_nonexistent_id() -> None:
    fake_experiment_id = 99999  # This experiment ID does not exist in the test api_server
    api_server_session = api.Session(
        master=f"{api_server.DEFAULT_HOST}:{api_server.DEFAULT_PORT}",
        user="user1",
        auth=None,
        cert=None,
    )
    with api_server.run_api_server(credentials=("user1", "password1", "token1"), ssl_keys=None):
        with pytest.raises(errors.NotFoundException):
            experiment.ExperimentReference(fake_experiment_id, api_server_session)
