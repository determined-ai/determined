import subprocess
from time import sleep
from typing import Dict, List, Tuple

import pytest

# from determined.experimental import Determined, ModelSortBy
from tests import config as conf
from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_job_queue_ahead_of(using_k8s: bool) -> None:
    if using_k8s:
        return
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    model = conf.tutorials_path("mnist_pytorch")
    for _ in range(4):
        exp.run_basic_test_with_temp_config(config, model, 1)

    jobs = JobInfo()
    ok = jobs.refresh_until_populated(10)
    assert ok

    ordered_ids = jobs.get_ids()
    subprocess.run(["det", "job", "update", ordered_ids[-1], "--ahead-of", ordered_ids[-2]])

    ordered_ids.insert(2, ordered_ids.pop(-1))
    assert jobs.check_order_equals(ordered_ids, 10)

    subprocess.run(["det", "job", "update-batch", f"{ordered_ids[-1]}.ahead-of={ordered_ids[-2]}"])

    ordered_ids.insert(2, ordered_ids.pop(-1))
    assert jobs.check_order_equals(ordered_ids, 10)


@pytest.mark.e2e_cpu
def test_job_queue_ahead_of_first(using_k8s: bool) -> None:
    if using_k8s:
        return
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    model = conf.tutorials_path("mnist_pytorch")
    for _ in range(4):
        exp.run_basic_test_with_temp_config(config, model, 1)

    jobs = JobInfo()
    ok = jobs.refresh_until_populated(10)
    assert ok

    ordered_ids = jobs.get_ids()
    subprocess.run(["det", "job", "update", ordered_ids[-1], "--ahead-of", ordered_ids[0]])

    ordered_ids.insert(0, ordered_ids.pop(-1))
    assert jobs.check_order_equals(ordered_ids, 10)

    subprocess.run(["det", "job", "update-batch", f"{ordered_ids[-1]}.ahead-of={ordered_ids[0]}"])

    ordered_ids.insert(0, ordered_ids.pop(-1))
    assert jobs.check_order_equals(ordered_ids, 10)


@pytest.mark.e2e_cpu
def test_job_queue_behind_of(using_k8s: bool) -> None:
    if using_k8s:
        return
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    model = conf.tutorials_path("mnist_pytorch")
    for _ in range(2):
        exp.run_basic_test_with_temp_config(config, model, 1)

    jobs = JobInfo()
    ok = jobs.refresh_until_populated(10)
    assert ok

    ordered_ids = jobs.get_ids()
    subprocess.run(["det", "job", "update", ordered_ids[0], "--behind-of", ordered_ids[1]])

    ordered_ids.insert(0, ordered_ids.pop(-1))
    assert jobs.check_order_equals(ordered_ids, 10)

    subprocess.run(["det", "job", "update-batch", f"{ordered_ids[0]}.behind-of={ordered_ids[1]}"])

    ordered_ids.insert(0, ordered_ids.pop(-1))
    assert jobs.check_order_equals(ordered_ids, 10)


@pytest.mark.e2e_cpu
def test_job_queue_behind_of_last(using_k8s: bool) -> None:
    if using_k8s:
        return
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    model = conf.tutorials_path("mnist_pytorch")
    for _ in range(2):
        exp.run_basic_test_with_temp_config(config, model, 1)

    jobs = JobInfo()
    ok = jobs.refresh_until_populated(10)
    assert ok

    ordered_ids = jobs.get_ids()
    subprocess.run(["det", "job", "update", ordered_ids[0], "--behind-of", ordered_ids[-1]])

    ordered_ids.append(ordered_ids.pop(0))
    assert jobs.check_order_equals(ordered_ids, 10)

    subprocess.run(["det", "job", "update-batch", f"{ordered_ids[0]}.behind-of={ordered_ids[-1]}"])

    ordered_ids.append(ordered_ids.pop(0))
    assert jobs.check_order_equals(ordered_ids, 10)


@pytest.mark.e2e_cpu
def test_job_queue_adjust_weight() -> None:
    config = conf.load_config(conf.tutorials_path("mnist_pytorch/const.yaml"))
    model = conf.tutorials_path("mnist_pytorch")
    for _ in range(2):
        exp.run_basic_test_with_temp_config(config, model, 1)

    jobs = JobInfo()
    ok = jobs.refresh_until_populated(10)
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

    return


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

    def refresh_until_populated(self, retries: int) -> bool:
        while retries > 0:
            retries -= 1
            if len(self.ids) > 0:
                return True
            sleep(0.5)
            self.refresh()
        return False

    def get_ids(self) -> List:
        return self.ids

    def get_job_weight(self, jobID: str) -> str:
        for value_dict in self.values:
            if value_dict["ID"] != jobID:
                continue
            return value_dict["Weight"]
        return ""

    def check_order_equals(self, expected: List, retries: int) -> bool:
        while retries > 0:
            self.refresh()
            if self.ids == expected:
                return True
            retries -= 1
            sleep(0.5)

        return False
