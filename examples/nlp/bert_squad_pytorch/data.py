from transformers.data.processors.squad import SquadV1Processor, SquadV2Processor
from transformers import squad_convert_examples_to_features
import urllib.request
import os


def load_and_cache_examples(
    data_dir: str,
    tokenizer,
    task,
    max_seq_length,
    doc_stride,
    max_query_length,
    evaluate=False,
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

    if not os.path.exists(data_dir):
        os.makedirs(data_dir)
    if evaluate:
        with urllib.request.urlopen(validation_url) as url:
            with open(data_dir + "/" + validation_file, "w") as f:
                f.write(url.read().decode())
        examples = processor.get_dev_examples(data_dir, filename=validation_file)
    else:
        with urllib.request.urlopen(train_url) as url:
            with open(data_dir + "/" + train_file, "w") as f:
                f.write(url.read().decode())
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
    return dataset, examples, features
