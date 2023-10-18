import numpy as np

import model_hub.huggingface as hf
import model_hub.utils as utils


def compute_metrics(
    data_config,
    column_names,
    post_processing_function,
    raw_datasets,
    tokenized_datasets,
    model,
    metric,
    predictions,
):
    inds, predictions = zip(*predictions)
    inds = np.hstack(inds)
    sorted_inds = np.argsort(inds)
    predictions = zip(*predictions)
    predictions = [utils.expand_like(p) for p in predictions]
    predictions = [p[sorted_inds] for p in predictions]

    # We need to add back in columns needed for validation.
    tokenized_datasets["validation"].set_format(
        type=tokenized_datasets["validation"].format["type"],
        columns=list(tokenized_datasets["validation"].features.keys()),
    )
    output = post_processing_function(
        examples=raw_datasets["validation"],
        features=tokenized_datasets["validation"],
        predictions=predictions,
        data_args=data_config,
        column_names=column_names,
        prefix="eval",
        model=model,
    )
    result = metric.compute(predictions=output.predictions, references=output.label_ids)
    # Then remove them again so that data collation doesn't break.
    hf.remove_unused_columns(model, tokenized_datasets["validation"])
    return result


class DatasetWithIndex:
    def __init__(self, dataset):
        self.dataset = dataset

    def __len__(self):
        return len(self.dataset)

    def __getitem__(self, idx):
        sample = self.dataset[idx]
        sample["ind"] = idx
        return sample
