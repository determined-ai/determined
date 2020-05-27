from transformers.data.processors.squad import SquadV1Processor, SquadV2Processor
from transformers import squad_convert_examples_to_features
import urllib
import json


def load_and_cache_examples(data_dir: str, tokenizer, task, max_seq_length, doc_stride, max_query_length, evaluate=False):
    if (task == "SQuAD1.1"):
        train_url = "https://rajpurkar.github.io/SQuAD-explorer/dataset/train-v1.1.json"
        validation_url = "https://rajpurkar.github.io/SQuAD-explorer/dataset/dev-v1.1.json"
        train_file = "train-v1.1.json"
        validation_file = "dev-v1.1.json"
        processor = SquadV1Processor()
    elif (task == "SQuAD2.0"):
        train_url = "https://rajpurkar.github.io/SQuAD-explorer/dataset/train-v2.0.json"
        validation_url = "https://rajpurkar.github.io/SQuAD-explorer/dataset/dev-v2.0.json"
        train_file = "train-v2.0.json"
        validation_file = "dev-v2.0.json"
        processor = SquadV2Processor()

    with urllib.request.urlopen(train_url) as url:
        train_data = json.loads(url.read().decode())
        with open(train_file, 'w') as f:
            json.dump(train_data, f)
    with urllib.request.urlopen(validation_url) as url:
        validation_data = json.loads(url.read().decode())
        with open(validation_file, 'w') as f:
            json.dump(validation_data, f)

    if evaluate:
        examples = processor.get_dev_examples(".", filename=validation_file)
    else:
        examples = processor.get_train_examples(".", filename=train_file)

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
