import pytest
import requests_mock

from determined.common import api
from determined.common.api import bindings
from determined.common.experimental import experiment
from tests.common import api_server


def test_wait_raises_exception_when_experiment_is_paused() -> None:
    session = api.Session(
        master=None,
        user=None,
        auth=None,
        cert=None,
    )
    experiment_id = 1
    expref = experiment.ExperimentReference(1, session)

    exp_resp = api_server.sample_get_experiment(
        id=experiment_id, state=bindings.experimentv1State.PAUSED
    )
    with requests_mock.Mocker() as m:
        m.get(
            f"/api/v1/experiments/{expref.id}",
            json=exp_resp.to_json(),
        )
        with pytest.raises(ValueError):
            expref.wait()
