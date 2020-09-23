import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly  # type: ignore
def test_bert_glue() -> None:
    config = conf.load_config(conf.nlp_examples_path("bert_glue_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("bert_glue_pytorch"), 1)


@pytest.mark.nightly  # type: ignore
def test_mnist_pytorch_multi_output() -> None:
    config = conf.load_config(conf.cv_examples_path("mnist_multi_output_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(
        config, conf.cv_examples_path("mnist_multi_output_pytorch"), 1
    )


@pytest.mark.nightly  # type: ignore
def test_imagenet_nas() -> None:
    config = conf.load_config(conf.nas_examples_path("gaea_pytorch/eval/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nas_examples_path("gaea_pytorch/eval"), 1)


@pytest.mark.nightly  # type: ignore
def test_gbt_estimator() -> None:
    config = conf.load_config(conf.decision_trees_examples_path("gbt_estimator/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(
        config, conf.decision_trees_examples_path("gbt_estimator"), 1
    )
