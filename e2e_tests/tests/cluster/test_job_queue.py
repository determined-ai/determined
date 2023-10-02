import subprocess
from time import sleep
from typing import Dict, List, Tuple

import pytest

# from determined.experimental import Determined, ModelSortBy
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_job_queue_adjust_weight() -> None:
    config = conf.tutorials_path("mnist_pytorch/const.yaml")
    model = conf.tutorials_path("mnist_pytorch")
    exp_ids = [exp.create_experiment(config, model) for _ in range(2)]

    try:
        jobs = JobInfo()
        ok = jobs.refresh_until_populated()
        assert ok

        ordered_ids = jobs.get_ids()
        subprocess.run(["det", "job", "update", ordered_ids[0], "--weight", "10"])

        sleep(2)
        jobs.refresh()
        new_weight = jobs.get_job_weight(ordered_ids[0])
        assert new_weight == "10"

        subprocess.run(["det", "job", "update-batch", f"{ordered_ids[1]}.weight=10"])

        sleep(2)
        jobs.refresh()
        new_weight = jobs.get_job_weight(ordered_ids[1])
        assert new_weight == "10"
    finally:
        exp.kill_experiments(exp_ids)


def get_raw_data() -> Tuple[List[Dict[str, str]], List[str]]:
    data = []
    ordered_ids = []
    output = subprocess.check_output(["det", "job", "list"]).decode("utf-8")
    lines = output.split("\n")
    keys = [line.strip() for line in lines[0].split("|")]

    for line in lines[2:]:
        line_dict = {}
        for i, field in enumerate(line.split("|")):
            if keys[i] == "ID":
                ordered_ids.append(field.strip())
            line_dict[keys[i]] = field.strip()
        data.append(line_dict)

    return data, ordered_ids


class JobInfo:
    def __init__(self) -> None:
        self.values, self.ids = get_raw_data()

    def refresh(self) -> None:
        self.values, self.ids = get_raw_data()

    def refresh_until_populated(self, retries: int = 10) -> bool:
        while retries > 0:
            retries -= 1
            if len(self.ids) > 0:
                return True
            sleep(0.5)
            self.refresh()
        print("self.ids remains empty")
        return False

    def get_ids(self) -> List:
        return self.ids

    def get_job_weight(self, jobID: str) -> str:
        for value_dict in self.values:
            if value_dict["ID"] != jobID:
                continue
            return value_dict["Weight"]
        return ""
