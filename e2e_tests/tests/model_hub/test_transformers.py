import os
import subprocess
from typing import Dict

import pytest

from tests import config as conf
from tests import experiment as exp


def set_docker_image(config: Dict) -> Dict:
    git_short_hash = subprocess.check_output(["git", "rev-parse", "--short", "HEAD"]).strip()
    git_short_hash = git_short_hash.decode("utf-8")

    config = conf.set_image(
        config, conf.TF1_CPU_IMAGE, f"determinedai/model-hub-transformers:{git_short_hash}"
    )
    return config


@pytest.mark.model_hub_transformers  # type: ignore
def test_token_classification_ner() -> None:
    example_path = conf.model_hub_examples_path("huggingface/token-classification")
    config = conf.load_config(os.path.join(example_path, "ner_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 32)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_token_classification_ner_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/token-classification")
    config = conf.load_config(os.path.join(example_path, "ner_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 32)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_language_modeling_clm() -> None:
    example_path = conf.model_hub_examples_path("huggingface/language-modeling")
    config = conf.load_config(os.path.join(example_path, "clm_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_language_modeling_clm_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/language-modeling")
    config = conf.load_config(os.path.join(example_path, "clm_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_language_modeling_mlm() -> None:
    example_path = conf.model_hub_examples_path("huggingface/language-modeling")
    config = conf.load_config(os.path.join(example_path, "mlm_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_language_modeling_mlm_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/language-modeling")
    config = conf.load_config(os.path.join(example_path, "mlm_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_language_modeling_plm() -> None:
    example_path = conf.model_hub_examples_path("huggingface/language-modeling")
    config = conf.load_config(os.path.join(example_path, "plm_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_language_modeling_plm_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/language-modeling")
    config = conf.load_config(os.path.join(example_path, "plm_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_multiple_choice_swag() -> None:
    example_path = conf.model_hub_examples_path("huggingface/multiple-choice")
    config = conf.load_config(os.path.join(example_path, "swag_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 64)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_multiple_choice_swag_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/multiple-choice")
    config = conf.load_config(os.path.join(example_path, "swag_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 64)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_text_classification_glue() -> None:
    example_path = conf.model_hub_examples_path("huggingface/text-classification")
    config = conf.load_config(os.path.join(example_path, "glue_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_text_classification_glue_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/text-classification")
    config = conf.load_config(os.path.join(example_path, "glue_config.yaml"))
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_text_classification_xnli() -> None:
    example_path = conf.model_hub_examples_path("huggingface/text-classification")
    config = conf.load_config(os.path.join(example_path, "xnli_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 128)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_text_classification_xnli_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/text-classification")
    config = conf.load_config(os.path.join(example_path, "xnli_config.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 128)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 64)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 64)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_with_beam_search() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad_beam_search.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_with_beam_search_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad_beam_search.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_v2() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad_v2.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 64)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_v2_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad_v2.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 64)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_v2_with_beam_search() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad_v2_beam_search.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)


@pytest.mark.model_hub_transformers  # type: ignore
def test_squad_v2_with_beam_search_amp() -> None:
    example_path = conf.model_hub_examples_path("huggingface/question-answering")
    config = conf.load_config(os.path.join(example_path, "squad_v2_beam_search.yaml"))
    config = conf.set_slots_per_trial(config, 8)
    config = conf.set_global_batch_size(config, 16)
    config = conf.set_max_length(config, {"batches": 200})
    config = conf.set_hparam(config, "use_apex_amp", True)
    config = set_docker_image(config)

    exp.run_basic_test_with_temp_config(config, example_path, 1)
