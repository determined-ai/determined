import os
import threading

import pytest

from tests import api_utils
from tests import config as conf
from tests import experiment as exp


@pytest.mark.parallel
@pytest.mark.timeout(300)
def test_gang_scheduling() -> None:
    sess = api_utils.user_session()
    total_slots = os.getenv("TOTAL_SLOTS")
    if total_slots is None:
        pytest.skip("test requires a static cluster and TOTAL_SLOTS set in the environment")

    config = conf.load_config(conf.tutorials_path("mnist_pytorch/distributed.yaml"))
    config = conf.set_slots_per_trial(config, int(total_slots))
    model = conf.tutorials_path("mnist_pytorch")

    def submit_job() -> None:
        ret_value = exp.run_basic_test_with_temp_config(sess, config, model, 1)
        print(ret_value)

    t = []
    for _i in range(2):
        t.append(threading.Thread(target=submit_job))
    for i in range(2):
        t[i].start()
    for i in range(2):
        t[i].join()
