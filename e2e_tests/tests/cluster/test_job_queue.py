import time
from typing import Dict, List, Tuple

import pytest

from determined.common import api

# from determined.experimental import Determined, ModelSortBy
from tests import api_utils
from tests import config as conf
from tests import detproc
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_job_queue_adjust_weight() -> None:
    sess = api_utils.user_session()
    config = conf.tutorials_path("mnist_pytorch/const.yaml")
    model = conf.tutorials_path("mnist_pytorch")
    exp_ids = [exp.create_experiment(sess, config, model) for _ in range(2)]

    try:
        jobs = JobInfo(sess)
        ok = jobs.refresh_until_populated()
        assert ok

        ordered_ids = jobs.get_ids()
        detproc.check_call(sess, ["det", "job", "update", ordered_ids[0], "--weight", "10"])

        time.sleep(2)
        jobs.refresh()
        new_weight = jobs.get_job_weight(ordered_ids[0])
        assert new_weight == "10"

        detproc.check_call(sess, ["det", "job", "update-batch", f"{ordered_ids[1]}.weight=10"])

        time.sleep(2)
        jobs.refresh()
        new_weight = jobs.get_job_weight(ordered_ids[1])
        assert new_weight == "10"
    finally:
        # Avoid leaking experiments even if this test fails.
        # Leaking experiments can block the cluster and other tests from running other tasks
        # while the experiments finish.
        exp.kill_experiments(sess, exp_ids)


def get_raw_data(sess: api.Session) -> Tuple[List[Dict[str, str]], List[str]]:
    data = []
    ordered_ids = []
    output = detproc.check_output(sess, ["det", "job", "list"])
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
    def __init__(self, sess: api.Session) -> None:
        self.sess = sess
        self.values, self.ids = get_raw_data(self.sess)

    def refresh(self) -> None:
        self.values, self.ids = get_raw_data(self.sess)

    def refresh_until_populated(self, retries: int = 10) -> bool:
        while retries > 0:
            retries -= 1
            if len(self.ids) > 0:
                return True
            time.sleep(0.5)
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
