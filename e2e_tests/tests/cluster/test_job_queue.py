import pytest

# from determined.experimental import Determined, ModelSortBy
# from tests import config as conf
# from tests import experiment as exp


@pytest.mark.e2e_cpu
def test_job_queue() -> None:
    return