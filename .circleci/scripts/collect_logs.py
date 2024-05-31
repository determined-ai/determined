#!/usr/bin/env python

import os
import warnings
from typing import List

import requests

from determined.common import api
from determined.common.api import certs


def obtain_token(username: str, password: str, master_address: str) -> str:
    """
    Gets a Determined token without using a Session.
    TODO: There is a replacement somewhere.
    """
    response = requests.post(
        f"{master_address}/api/v1/auth/login",
        json={"username": username, "password": password},
        verify=False,
    )
    response.raise_for_status()
    token = response.json()["token"]
    assert isinstance(token, str)
    return token


class CliBase:
    """
    Developer-only CLI.
    """

    def __init__(self, username: str, password: str, mlde_host: str = "http://localhost:8080"):
        self.username = username
        self.password = password
        self.mlde_host = mlde_host
        token = obtain_token(username, password, master_address=mlde_host)
        self.token = token
        cert = certs.Cert(noverify=True)
        session = api.Session(mlde_host, username=username, token=token, cert=cert)
        self.session = session


warnings.filterwarnings("ignore", category=FutureWarning, module="determined.*")
from determined.common.api import bindings as b


class Cli(CliBase):
    def get_experiments(self, just_active: bool = True) -> List[b.v1Experiment]:
        non_terminal_states = [
            b.experimentv1State.ACTIVE,
            b.experimentv1State.PAUSED,
            b.experimentv1State.RUNNING,
        ]
        states = non_terminal_states if just_active else None
        resp = b.get_GetExperiments(self.session, archived=False, states=states)
        experiments = [exp for exp in resp.experiments]
        return experiments

    def get_experiments_ids(self, just_active: bool = True) -> List[int]:
        experiments = self.get_experiments(just_active)
        return [c.id for c in experiments]

    def get_single_trial_exp_logs(self, exp_id: int):
        """
        Get logs for a single trial experiment.
        """
        trials = b.get_GetExperimentTrials(self.session, experimentId=exp_id).trials
        t_id = trials[0].id
        logs = b.get_TrialLogs(self.session, trialId=t_id)
        for log in logs:
            yield log.message

    def save_all_logs(self, output_dir: str, just_active: bool = False):
        """
        Save all logs for all experiments to a given directory
        """
        experiments = self.get_experiments(just_active)
        output_dir = output_dir.rstrip("/")
        os.makedirs(output_dir, exist_ok=True)
        for c in experiments:
            output = f"{output_dir}/{c.id}-{c.name}.log"
            print(f"Saving logs {output}")
            with open(output, "w") as f:
                for log in self.get_single_trial_exp_logs(c.id):
                    f.write(log)


if __name__ == "__main__":
    import fire

    fire.Fire(Cli)
