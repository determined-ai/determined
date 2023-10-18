"""
The data processing functions in this file are largely derived from the example for question
answering in the huggingface transformers library. The license for the transformer's library is
reproduced below.

==================================================================================================

Copyright 2020 The HuggingFace Team. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
"""
import collections
import json
import logging
import os
from typing import Optional, Tuple

import numpy as np
import transformers

logger = logging.getLogger(__name__)


def prepare_features(
    split,
    data_args,
    tokenizer,
    column_names,
    examples,
):
    question_column_name = "question" if "question" in column_names else column_names[0]
    context_column_name = "context" if "context" in column_names else column_names[1]
    answer_column_name = "answers" if "answers" in column_names else column_names[2]

    # Padding side determines if we do (question|context) or (context|question).
    pad_on_right = tokenizer.padding_side == "right"

    if data_args.max_seq_length > tokenizer.model_max_length:
        logger.warning(
            f"The max_seq_length passed ({data_args.max_seq_length}) is larger than the maximum "
            f"length for the model ({tokenizer.model_max_length}). Using "
            f"max_seq_length={tokenizer.model_max_length}."
        )
    max_seq_length = min(data_args.max_seq_length, tokenizer.model_max_length)

    if split == "train":
        # Tokenize our examples with truncation and maybe padding, but keep the overflows using a
        # stride. This results in one example possible giving several features when a context is
        # long, each of those features having a context that overlaps a bit the context of the
        # previous feature.
        tokenized_examples = tokenizer(
            examples[question_column_name if pad_on_right else context_column_name],
            examples[context_column_name if pad_on_right else question_column_name],
            truncation="only_second" if pad_on_right else "only_first",
            max_length=max_seq_length,
            stride=data_args.doc_stride,
            return_overflowing_tokens=True,
            return_offsets_mapping=True,
            return_special_tokens_mask=True,
            return_token_type_ids=True,
            padding="max_length",
        )

        # Since one example might give us several features if it has a long context, we need a map
        # from a feature to its corresponding example. This key gives us just that.
        sample_mapping = tokenized_examples.pop("overflow_to_sample_mapping")
        # The offset mappings will give us a map from token to character position in the original
        # context. This will help us compute the start_positions and end_positions.
        offset_mapping = tokenized_examples.pop("offset_mapping")
        # The special tokens will help us build the p_mask (which indicates the tokens that can't
        # be in answers).
        special_tokens = tokenized_examples.pop("special_tokens_mask")

        # Let's label those examples!
        tokenized_examples["start_positions"] = []
        tokenized_examples["end_positions"] = []
        tokenized_examples["is_impossible"] = []
        tokenized_examples["cls_index"] = []
        tokenized_examples["p_mask"] = []

        for i, offsets in enumerate(offset_mapping):
            # We will label impossible answers with the index of the CLS token.
            input_ids = tokenized_examples["input_ids"][i]
            cls_index = input_ids.index(tokenizer.cls_token_id)
            tokenized_examples["cls_index"].append(cls_index)

            # Grab the sequence corresponding to that example (to know what is the context and
            # what is the question).
            sequence_ids = tokenized_examples["token_type_ids"][i]
            for k, s in enumerate(special_tokens[i]):
                if s:
                    sequence_ids[k] = 3
            context_idx = 1 if pad_on_right else 0

            # Build the p_mask: non special tokens and context gets 0.0, the others get 1.0.
            # The cls token gets 1.0 too (for predictions of empty answers).
            tokenized_examples["p_mask"].append(
                [
                    0.0
                    if (not special_tokens[i][k] and s == context_idx) or k == cls_index
                    else 1.0
                    for k, s in enumerate(sequence_ids)
                ]
            )

            # One example can give several spans, this is the index of the example containing
            # this span of text.
            sample_index = sample_mapping[i]
            answers = examples[answer_column_name][sample_index]
            # If no answers are given, set the cls_index as answer.
            if len(answers["answer_start"]) == 0:
                tokenized_examples["start_positions"].append(cls_index)
                tokenized_examples["end_positions"].append(cls_index)
                tokenized_examples["is_impossible"].append(1.0)
            else:
                # Start/end character index of the answer in the text.
                start_char = answers["answer_start"][0]
                end_char = start_char + len(answers["text"][0])

                # Start token index of the current span in the text.
                token_start_index = 0
                while sequence_ids[token_start_index] != context_idx:
                    token_start_index += 1

                # End token index of the current span in the text.
                token_end_index = len(input_ids) - 1
                while sequence_ids[token_end_index] != context_idx:
                    token_end_index -= 1
                # Detect if the answer is out of the span (in which case this feature is labeled
                # with the CLS index).
                if not (
                    offsets[token_start_index][0] <= start_char
                    and offsets[token_end_index][1] >= end_char
                ):
                    tokenized_examples["start_positions"].append(cls_index)
                    tokenized_examples["end_positions"].append(cls_index)
                    tokenized_examples["is_impossible"].append(1.0)
                else:
                    # Otherwise move the token_start_index and token_end_index to the two ends of
                    # the answer. Note: we could go after the last offset if the answer is the
                    # last word (edge case).
                    while (
                        token_start_index < len(offsets)
                        and offsets[token_start_index][0] <= start_char
                    ):
                        token_start_index += 1
                    tokenized_examples["start_positions"].append(token_start_index - 1)
                    while offsets[token_end_index][1] >= end_char:
                        token_end_index -= 1
                    tokenized_examples["end_positions"].append(token_end_index + 1)
                    tokenized_examples["is_impossible"].append(0.0)
    else:  # preprocessing for validation
        # Tokenize our examples with truncation and maybe padding, but keep the overflows using
        # a stride. This results in one example possible giving several features when a context is
        # long, each of those features having a context that overlaps a bit the context of the
        # previous feature.
        tokenized_examples = tokenizer(
            examples[question_column_name if pad_on_right else context_column_name],
            examples[context_column_name if pad_on_right else question_column_name],
            truncation="only_second" if pad_on_right else "only_first",
            max_length=max_seq_length,
            stride=data_args.doc_stride,
            return_overflowing_tokens=True,
            return_offsets_mapping=True,
            return_special_tokens_mask=True,
            return_token_type_ids=True,
            padding="max_length",
        )

        # Since one example might give us several features if it has a long context, we need a map
        # from a feature to its corresponding example. This key gives us just that.
        sample_mapping = tokenized_examples.pop("overflow_to_sample_mapping")

        # The special tokens will help us build the p_mask (which indicates the tokens that can't
        # be in answers).
        special_tokens = tokenized_examples.pop("special_tokens_mask")

        # For evaluation, we will need to convert our predictions to substrings of the context, so
        # we keep the corresponding example_id and we will store the offset mappings.
        tokenized_examples["example_id"] = []

        # We still provide the index of the CLS token and the p_mask to the model, but not the
        # is_impossible label.
        tokenized_examples["cls_index"] = []
        tokenized_examples["p_mask"] = []

        for i, input_ids in enumerate(tokenized_examples["input_ids"]):
            # Find the CLS token in the input ids.
            cls_index = input_ids.index(tokenizer.cls_token_id)
            tokenized_examples["cls_index"].append(cls_index)

            # Grab the sequence corresponding to that example (to know what is the context and
            # what is the question).
            sequence_ids = tokenized_examples["token_type_ids"][i]
            for k, s in enumerate(special_tokens[i]):
                if s:
                    sequence_ids[k] = 3
            context_idx = 1 if pad_on_right else 0

            # Build the p_mask: non special tokens and context gets 0.0, the others 1.0.
            tokenized_examples["p_mask"].append(
                [
                    0.0
                    if (not special_tokens[i][k] and s == context_idx) or k == cls_index
                    else 1.0
                    for k, s in enumerate(sequence_ids)
                ]
            )

            # One example can give several spans, this is the index of the example containing this
            # span of text.
            sample_index = sample_mapping[i]
            tokenized_examples["example_id"].append(examples["id"][sample_index])

            # Set to None the offset_mapping that are not part of the context so it's easy to
            # determine if a token position is part of the context or not.
            tokenized_examples["offset_mapping"][i] = [
                (o if sequence_ids[k] == context_idx else None)
                for k, o in enumerate(tokenized_examples["offset_mapping"][i])
            ]
    return tokenized_examples


def postprocess_qa_predictions_with_beam_search(
    examples,
    features,
    predictions: Tuple[np.ndarray, np.ndarray],
    version_2_with_negative: bool = False,
    n_best_size: int = 20,
    max_answer_length: int = 30,
    start_n_top: int = 5,
    end_n_top: int = 5,
    output_dir: Optional[str] = None,
    prefix: Optional[str] = None,
):
    """
    Post-processes the predictions of a question-answering model with beam search to convert them
    to answers that are substrings of the original contexts. This is the postprocessing functions
    for models that return start and end logits, indices, as well as cls token predictions.
    Args:
        examples: The non-preprocessed dataset (see the main script for more information).
        features: The processed dataset (see the main script for more information).
        predictions (:obj:`Tuple[np.ndarray, np.ndarray]`):
            The predictions of the model: two arrays containing the start logits and the end logits
            respectively. Its first dimension must match the number of elements of :obj:`features`.
        version_2_with_negative (:obj:`bool`, `optional`, defaults to :obj:`False`):
            Whether or not the underlying dataset contains examples with no answers.
        n_best_size (:obj:`int`, `optional`, defaults to 20):
            The total number of n-best predictions to generate when looking for an answer.
        max_answer_length (:obj:`int`, `optional`, defaults to 30):
            The maximum length of an answer that can be generated. This is needed because the start
            and end predictions are not conditioned on one another.
        start_n_top (:obj:`int`, `optional`, defaults to 5):
            The number of top start logits too keep when searching for the :obj:`n_best_size`
            predictions.
        end_n_top (:obj:`int`, `optional`, defaults to 5):
            The number of top end logits too keep when searching for the :obj:`n_best_size`
            predictions.
        output_dir (:obj:`str`, `optional`):
            If provided, the dictionaries of predictions, n_best predictions (with their scores
            and logits) and, if :obj:`version_2_with_negative=True`, the dictionary of the scores
            differences between best and null answers, are saved in `output_dir`.
        prefix (:obj:`str`, `optional`):
            If provided, the dictionaries mentioned above are saved with `prefix` added to their
            names.
    """
    assert len(predictions) == 5, "`predictions` should be a tuple with five elements."
    (
        start_top_log_probs,
        start_top_index,
        end_top_log_probs,
        end_top_index,
        cls_logits,
    ) = predictions

    assert len(predictions[0]) == len(
        features
    ), f"Got {len(predictions[0])} predicitions and {len(features)} features."

    # Build a map example to its corresponding features.
    example_id_to_index = {k: i for i, k in enumerate(examples["id"])}
    features_per_example = collections.defaultdict(list)
    for i, feature in enumerate(features):
        features_per_example[example_id_to_index[feature["example_id"]]].append(i)

    # The dictionaries we have to fill.
    all_predictions = collections.OrderedDict()
    all_nbest_json = collections.OrderedDict()
    scores_diff_json = collections.OrderedDict() if version_2_with_negative else None

    # Logging.
    logger.info(
        f"Post-processing {len(examples)} example predictions split into {len(features)} features."
    )

    # Let's loop over all the examples!
    for example_index, example in enumerate(examples):
        # Those are the indices of the features associated to the current example.
        feature_indices = features_per_example[example_index]

        min_null_score = None
        prelim_predictions = []

        # Looping through all the features associated to the current example.
        for feature_index in feature_indices:
            # We grab the predictions of the model for this feature.
            start_log_prob = start_top_log_probs[feature_index]
            start_indexes = start_top_index[feature_index]
            end_log_prob = end_top_log_probs[feature_index]
            end_indexes = end_top_index[feature_index]
            feature_null_score = cls_logits[feature_index]
            # This is what will allow us to map some the positions in our logits to span of texts
            # in the original context.
            offset_mapping = features[feature_index]["offset_mapping"]
            # Optional `token_is_max_context`, if provided we will remove answers that do not have
            # the maximum context available in the current feature.
            token_is_max_context = features[feature_index].get("token_is_max_context", None)

            # Update minimum null prediction
            if min_null_score is None or feature_null_score < min_null_score:
                min_null_score = feature_null_score

            # Go through all possibilities for the `n_start_top`/`n_end_top` greater start and
            # end logits.
            for i in range(start_n_top):
                for j in range(end_n_top):
                    start_index = int(start_indexes[i])
                    j_index = i * end_n_top + j
                    end_index = int(end_indexes[j_index])
                    # Don't consider out-of-scope answers (last part of the test should be
                    # unnecessary because of the p_mask but let's not take any risk)
                    if (
                        start_index >= len(offset_mapping)
                        or end_index >= len(offset_mapping)
                        or offset_mapping[start_index] is None
                        or offset_mapping[end_index] is None
                    ):
                        continue
                    # Don't consider answers with a length negative or > max_answer_length.
                    if end_index < start_index or end_index - start_index + 1 > max_answer_length:
                        continue
                    # Don't consider answer that don't have the maximum context available (if such
                    # information is provided).
                    if token_is_max_context is not None and not token_is_max_context.get(
                        str(start_index), False
                    ):
                        continue
                    prelim_predictions.append(
                        {
                            "offsets": (
                                offset_mapping[start_index][0],
                                offset_mapping[end_index][1],
                            ),
                            "score": start_log_prob[i] + end_log_prob[j_index],
                            "start_log_prob": start_log_prob[i],
                            "end_log_prob": end_log_prob[j_index],
                        }
                    )

        # Only keep the best `n_best_size` predictions.
        predictions = sorted(prelim_predictions, key=lambda x: x["score"], reverse=True)[
            :n_best_size
        ]

        # Use the offsets to gather the answer text in the original context.
        context = example["context"]
        for pred in predictions:
            offsets = pred.pop("offsets")
            pred["text"] = context[offsets[0] : offsets[1]]

        # In the very rare edge case we have not a single non-null prediction, we create a fake
        # prediction to avoid failure.
        if len(predictions) == 0:
            predictions.insert(
                0, {"text": "", "start_logit": -1e-6, "end_logit": -1e-6, "score": -2e-6}
            )

        # Compute the softmax of all scores (we do it with numpy to stay independent from torch/tf
        # in this file, using the LogSumExp trick).
        scores = np.array([pred.pop("score") for pred in predictions])
        exp_scores = np.exp(scores - np.max(scores))
        probs = exp_scores / exp_scores.sum()

        # Include the probabilities in our predictions.
        for prob, pred in zip(probs, predictions):
            pred["probability"] = prob

        # Pick the best prediction and set the probability for the null answer.
        all_predictions[example["id"]] = predictions[0]["text"]
        if version_2_with_negative:
            scores_diff_json[example["id"]] = float(min_null_score)

        # Make `predictions` JSON-serializable by casting np.float back to float.
        all_nbest_json[example["id"]] = [
            {
                k: (float(v) if isinstance(v, (np.float16, np.float32, np.float64)) else v)
                for k, v in pred.items()
            }
            for pred in predictions
        ]
        all_nbest_json[example["id"]] = [
            {
                k: (int(v) if isinstance(v, (np.int16, np.int32, np.int64)) else v)
                for k, v in pred.items()
            }
            for pred in predictions
        ]

    # If we have an output_dir, let's save all those dicts.
    if output_dir is not None:
        assert os.path.isdir(output_dir), f"{output_dir} is not a directory."

        prediction_file = os.path.join(
            output_dir, "predictions.json" if prefix is None else f"{prefix}_predictions.json"
        )
        nbest_file = os.path.join(
            output_dir,
            "nbest_predictions.json" if prefix is None else f"{prefix}_nbest_predictions.json",
        )
        if version_2_with_negative:
            null_odds_file = os.path.join(
                output_dir, "null_odds.json" if prefix is None else f"{prefix}_null_odds.json"
            )

        logger.info(f"Saving predictions to {prediction_file}.")
        with open(prediction_file, "w") as writer:
            writer.write(json.dumps(all_predictions, indent=4) + "\n")
        logger.info(f"Saving nbest_preds to {nbest_file}.")
        with open(nbest_file, "w") as writer:
            writer.write(json.dumps(all_nbest_json, indent=4) + "\n")
        if version_2_with_negative:
            logger.info(f"Saving null_odds to {null_odds_file}.")
            with open(null_odds_file, "w") as writer:
                writer.write(json.dumps(scores_diff_json, indent=4) + "\n")

    return all_predictions, scores_diff_json


def post_processing_function(
    examples,
    features,
    predictions,
    data_args,
    column_names,
    prefix,
    model,
):
    # Post-processing: we match the start logits and end logits to answers in the original context.
    answer_column_name = "answers" if "answers" in column_names else column_names[2]
    predictions, scores_diff_json = postprocess_qa_predictions_with_beam_search(
        examples=examples,
        features=features,
        predictions=predictions,
        version_2_with_negative=data_args.version_2_with_negative,
        n_best_size=data_args.n_best_size,
        max_answer_length=data_args.max_answer_length,
        start_n_top=model.config.start_n_top,
        end_n_top=model.config.end_n_top,
        output_dir=data_args.output_dir,
        prefix=prefix,
    )
    # Format the result to the format the metric expects.
    if data_args.version_2_with_negative:
        formatted_predictions = [
            {"id": k, "prediction_text": v, "no_answer_probability": scores_diff_json[k]}
            for k, v in predictions.items()
        ]
    else:
        formatted_predictions = [{"id": k, "prediction_text": v} for k, v in predictions.items()]

    references = [{"id": ex["id"], "answers": ex[answer_column_name]} for ex in examples]
    return transformers.EvalPrediction(predictions=formatted_predictions, label_ids=references)
