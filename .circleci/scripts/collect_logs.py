#!/usr/bin/env python

import os
import warnings
from typing import List, Union

import requests

from determined.common import api
from determined.common.api import certs

Task = Union["b.v1Command", "b.v1Notebook", "b.v1Shell", "b.v1Tensorboard"]


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

    def get_trial_logs(self, trial_id: int):
        logs = b.get_TrialLogs(self.session, trialId=trial_id)
        for log in logs:
            yield log.message

    def get_single_trial_exp_logs(self, exp_id: int):
        """
        Get logs for a single trial experiment.
        """
        trials = b.get_GetExperimentTrials(self.session, experimentId=exp_id).trials
        t_id = trials[0].id
        return self.get_trial_logs(t_id)

    def save_single_trial_experiment_logs(self, output_dir: str, just_active: bool = False):
        """
        Save logs from first trial of each experiment to a given directory.
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

    def get_tasks(self) -> List[Task]:
        tasks: List[Task] = []
        tasks.extend(b.get_GetCommands(self.session).commands)
        tasks.extend(b.get_GetNotebooks(self.session).notebooks)
        tasks.extend(b.get_GetShells(self.session).shells)
        tasks.extend(b.get_GetTensorboards(self.session).tensorboards)
        return tasks

    def clean_os_path(self, s: str) -> str:
        """
        remove some characters and replace spaces with underscores
        """
        return "".join([c if (c.isalnum() or c in "/-_.") else "_" for c in s])

    def save_all_logs(self, output_dir: str, just_active: bool = False):
        """
        Save all task? logs to a given directory
        """
        experiments = self.get_experiments(just_active)
        output_dir = output_dir.rstrip("/")
        os.makedirs(output_dir, exist_ok=True)
        for exp in experiments:
            trials = b.get_GetExperimentTrials(self.session, experimentId=exp.id).trials
            for t in trials:
                output = self.clean_os_path(f"{output_dir}/exp{exp.id}-trial{t.id}-{exp.name}.log")
                print(f"Saving {output}")
                with open(output, "w") as f:
                    for log in self.get_trial_logs(t.id):
                        f.write(log)
        tasks = self.get_tasks()
        for task in tasks:
            output = self.clean_os_path(f"{output_dir}/task-{task.id}-{task.description}.log")
            print(f"Saving {output}")
            with open(output, "w") as f:
                for log in b.get_TaskLogs(self.session, taskId=task.id):
                    f.write(log.message)


if __name__ == "__main__":
    import fire

    fire.Fire(Cli)
