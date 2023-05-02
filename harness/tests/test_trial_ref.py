"""Test functionality in the trial module of the SDK."""
import pytest

from determined.common import api
from determined.common.api import errors
from determined.common.experimental import trial
from tests.common import api_server


def test_trial_ref_createable_on_extant_id() -> None:
    real_trial_id = 1  # This trial ID exists in the test api_server
    api_server_session = api.Session(
        master=f"{api_server.DEFAULT_HOST}:{api_server.DEFAULT_PORT}",
        user="user1",
        auth=None,
        cert=None,
    )
    with api_server.run_api_server(credentials=("user1", "password1", "token1"), ssl_keys=None):
        trial.TrialReference(real_trial_id, api_server_session)


def test_trial_ref_not_createable_on_nonexistent_id() -> None:
    fake_trial_id = 99999  # This trial ID does not exist in the test api_server
    api_server_session = api.Session(
        master=f"{api_server.DEFAULT_HOST}:{api_server.DEFAULT_PORT}",
        user="user1",
        auth=None,
        cert=None,
    )
    with api_server.run_api_server(credentials=("user1", "password1", "token1"), ssl_keys=None):
        with pytest.raises(errors.NotFoundException):
            trial.TrialReference(fake_trial_id, api_server_session)
