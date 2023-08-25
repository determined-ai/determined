import pytest

from tests import config as conf
from tests import experiment as exp


@pytest.mark.nightly
def test_bert_glue_const() -> None:
    config = conf.load_config(conf.nlp_examples_path("bert_glue_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("bert_glue_pytorch"), 1)


@pytest.mark.nightly
def test_gaea_pytorch_const() -> None:
    config = conf.load_config(conf.nas_examples_path("gaea_pytorch/eval/const.yaml"))
    config = conf.set_global_batch_size(config, 32)
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.nas_examples_path("gaea_pytorch/eval"), 1)


@pytest.mark.nightly
def test_gan_mnist_pytorch_const() -> None:
    config = conf.load_config(conf.gan_examples_path("gan_mnist_pytorch/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path("gan_mnist_pytorch"), 1)


@pytest.mark.nightly
def test_pix2pix_facades_const() -> None:
    config = conf.load_config(conf.gan_examples_path("pix2pix_tf_keras/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})

    exp.run_basic_test_with_temp_config(config, conf.gan_examples_path("pix2pix_tf_keras"), 1)


@pytest.mark.nightly
def test_word_language_transformer_const() -> None:
    config = conf.load_config(conf.nlp_examples_path("word_language_model/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = config.copy()
    config["hyperparameters"]["model_cls"] = "Transformer"

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("word_language_model"), 1)


@pytest.mark.nightly
def test_word_language_lstm_const() -> None:
    config = conf.load_config(conf.nlp_examples_path("word_language_model/const.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = config.copy()
    config["hyperparameters"]["model_cls"] = "LSTM"
    config["hyperparameters"]["tied"] = False

    exp.run_basic_test_with_temp_config(config, conf.nlp_examples_path("word_language_model"), 1)


@pytest.mark.nightly
def test_protein_pytorch_geometric() -> None:
    config = conf.load_config(conf.graphs_examples_path("proteins_pytorch_geometric/const.yaml"))

    exp.run_basic_test_with_temp_config(
        config, conf.graphs_examples_path("proteins_pytorch_geometric"), 1
    )
