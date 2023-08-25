import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly
def test_protein_pytorch_geometric() -> None:
    config = conf.load_config(conf.graphs_examples_path("proteins_pytorch_geometric/const.yaml"))

    exp.run_basic_test_with_temp_config(
        config, conf.graphs_examples_path("proteins_pytorch_geometric"), 1
    )
