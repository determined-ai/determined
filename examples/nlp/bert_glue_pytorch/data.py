import os

import torch
from torch.utils.data import TensorDataset

from transformers import AutoTokenizer
from datasets import load_dataset

def load_and_cache_examples(config, max_seq_length, evaluate=False):
    task = config["task"].lower()

    tokenizer = AutoTokenizer.from_pretrained(
        config["model_name_or_path"], do_lower_case=True, cache_dir=None
    )

    print("Creating features from dataset")
    raw_dataset = load_dataset("glue", task)
    label_list = raw_dataset["train"].features["label"].names

    def encode(examples):
        return tokenizer(examples['sentence1'], examples['sentence2'], truncation=True, padding='max_length', max_length=max_seq_length)

    split = "test" if evaluate else "train"
    features = raw_dataset[split].map(encode, batched=True)

    # Convert to Tensors and build dataset
    all_input_ids = torch.tensor([f['input_ids'] for f in features], dtype=torch.long)
    all_attention_mask = torch.tensor([f['attention_mask'] for f in features], dtype=torch.long)
    all_token_type_ids = torch.tensor([f['token_type_ids'] for f in features], dtype=torch.long)
    all_labels = torch.tensor([f['label'] for f in features], dtype=torch.long)

    dataset = TensorDataset(all_input_ids, all_attention_mask, all_token_type_ids, all_labels)
    return dataset
