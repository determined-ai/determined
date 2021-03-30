from transformers.data.processors.squad import (
    SquadV1Processor,
    SquadV2Processor,
    SquadFeatures,
)
from transformers import squad_convert_examples_to_features
import urllib.request
import os
import torch

from pathlib import Path


def data_directory(using_bind_mount: bool, rank: int, bind_mount_path: Path = None):
    base_dir = bind_mount_path if using_bind_mount else Path("/tmp")
    return base_dir / f"data-rank{rank}"


def cache_dir(using_bind_mount: bool, rank: int, bind_mount_path: Path = None):
    base_dir = bind_mount_path if using_bind_mount else Path("/tmp")
    return base_dir / f"cache/{rank}"


def load_and_cache_examples(
    data_dir: Path,
    tokenizer,
    task,
    max_seq_length,
    doc_stride,
    max_query_length,
    evaluate=False,
    model_name=None,
):
    if task == "SQuAD1.1":
        train_url = "https://rajpurkar.github.io/SQuAD-explorer/dataset/train-v1.1.json"
        validation_url = (
            "https://rajpurkar.github.io/SQuAD-explorer/dataset/dev-v1.1.json"
        )
        train_file = "train-v1.1.json"
        validation_file = "dev-v1.1.json"
        processor = SquadV1Processor()
    elif task == "SQuAD2.0":
        train_url = "https://rajpurkar.github.io/SQuAD-explorer/dataset/train-v2.0.json"
        validation_url = (
            "https://rajpurkar.github.io/SQuAD-explorer/dataset/dev-v2.0.json"
        )
        train_file = "train-v2.0.json"
        validation_file = "dev-v2.0.json"
        processor = SquadV2Processor()
    else:
        raise NameError("Incompatible dataset detected")

    if not data_dir.exists():
        data_dir.mkdir(parents=True)
    if evaluate:
        # TODO: Cache instead of always downloading
        with urllib.request.urlopen(validation_url) as url:
            val_path = data_dir / validation_file
            with val_path.open("w") as f:
                f.write(url.read().decode())

    else:
        with urllib.request.urlopen(train_url) as url:
            train_path = data_dir / train_file
            with train_path.open("w") as f:
                f.write(url.read().decode())

    # Load data features from cache or dataset file
    cached_features_file = os.path.join(
        str(data_dir.absolute()),
        "cache_{}_{}".format(
            "dev" if evaluate else "train",
            model_name,
        ),
    )

    # Init features and dataset from cache if it exists
    overwrite_cache = (
        False  # Set to True to do a cache wipe (TODO: Make cache wipe configurable)
    )
    if os.path.exists(cached_features_file) and not overwrite_cache:
        print("Loading features from cached file %s", cached_features_file)
        features_and_dataset = torch.load(cached_features_file)
        features, dataset, examples = (
            features_and_dataset["features"],
            features_and_dataset["dataset"],
            features_and_dataset["examples"],
        )
    else:
        if evaluate:
            examples = processor.get_dev_examples(data_dir, filename=validation_file)
        else:
            examples = processor.get_train_examples(data_dir, filename=train_file)
        features, dataset = squad_convert_examples_to_features(
            examples=examples,
            tokenizer=tokenizer,
            max_seq_length=max_seq_length,
            doc_stride=doc_stride,
            max_query_length=max_query_length,
            is_training=not evaluate,
            return_dataset="pt",
        )
        print("Saving features into cached file %s", cached_features_file)
        torch.save(
            {"features": features, "dataset": dataset, "examples": examples},
            cached_features_file,
        )
    return dataset, examples, features
