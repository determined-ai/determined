import os

import torch
from torch.utils.data import TensorDataset

from transformers import (
    glue_convert_examples_to_features as convert_examples_to_features,
)
from transformers import glue_output_modes as output_modes
from transformers import glue_processors as processors

import constants
import download_glue_data


def download_data(task, data_dir, path_to_mrpc) -> None:
    if (
        os.path.exists(data_dir + "/" + task)
        and len(os.listdir(data_dir + "/" + task)) > 0
    ):
        # Exit if the data already exists
        return

    if not os.path.exists(data_dir):
        os.mkdir(data_dir)

    if task == "MRPC":
        download_glue_data.format_mrpc(data_dir, path_to_mrpc)
    elif task == "diagnostic":
        download_glue_data.download_diagnostic(data_dir)
    else:
        download_glue_data.download_and_extract(task, data_dir)


def load_and_cache_examples(
    base_data_dir: str, config, model_type, max_seq_length, evaluate=False
):
    model_type = model_type.lower()
    task = config["task"].lower()
    data_dir = f"{base_data_dir}/{task.upper()}"

    config_class, model_class, tokenizer_class = constants.MODEL_CLASSES[model_type]
    tokenizer = tokenizer_class.from_pretrained(
        config["model_name_or_path"], do_lower_case=True, cache_dir=None
    )

    processor = processors[task]()
    output_mode = output_modes[task]

    print("Creating features from dataset file at %s", data_dir)
    label_list = processor.get_labels()
    if task in ["mnli", "mnli-mm"] and model_type in ["roberta"]:
        # HACK(label indices are swapped in RoBERTa pretrained model)
        label_list[1], label_list[2] = label_list[2], label_list[1]
    examples = (
        processor.get_dev_examples(data_dir)
        if evaluate
        else processor.get_train_examples(data_dir)
    )
    features = convert_examples_to_features(
        examples,
        tokenizer,
        label_list=label_list,
        max_length=max_seq_length,
        output_mode=output_mode,
    )

    # Convert to Tensors and build dataset
    all_input_ids = torch.tensor([f.input_ids for f in features], dtype=torch.long)
    all_attention_mask = torch.tensor(
        [f.attention_mask for f in features], dtype=torch.long
    )
    all_token_type_ids = torch.tensor(
        [f.token_type_ids for f in features], dtype=torch.long
    )
    all_labels = torch.tensor([f.label for f in features], dtype=torch.long)

    dataset = TensorDataset(
        all_input_ids, all_attention_mask, all_token_type_ids, all_labels
    )
    return dataset
