from transformers.data.processors.squad import SquadV1Processor
from transformers import squad_convert_examples_to_features
import tensorflow_datasets as tfds


def load_and_cache_examples(data_dir: str, tokenizer, model_type, max_seq_length, doc_stride, max_query_length, model_name_or_path, evaluate=False):
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
    return dataset, examples, features
