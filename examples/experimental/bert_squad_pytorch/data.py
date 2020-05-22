import os
import constants
import torch
from transformers.data.processors.squad import SquadV1Processor
from transformers import squad_convert_examples_to_features

def load_and_cache_examples(data_dir: str, config, model_type, max_seq_length, doc_stride, max_query_length, model_name_or_path, evaluate=False, output_examples=False):
    config_class, tokenizer_class, model_class = constants.MODEL_CLASSES[model_type]
    tokenizer = tokenizer_class.from_pretrained(
        config["model_name_or_path"], do_lower_case=True, cache_dir=None
    )
    try:
        import tensorflow_datasets as tfds
    except ImportError:
        raise ImportError("Tensorflow_datasets needs to be installed.")
    tfds_examples = tfds.load("squad", data_dir=data_dir)
    examples = SquadV1Processor().get_examples_from_dataset(tfds_examples, evaluate=evaluate)
    features, dataset = squad_convert_examples_to_features(
            examples=examples,
            tokenizer=tokenizer,
            max_seq_length=max_seq_length,
            doc_stride=doc_stride,
            max_query_length=max_query_length,
            is_training=not evaluate,
            return_dataset="pt",
    )

    if output_examples:
        return dataset, examples, features
    return dataset
