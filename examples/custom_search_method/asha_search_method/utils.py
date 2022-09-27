from typing import Dict
import random


############################################################################
# User-defined function that generates hyperparameters for each trial.
# The hyperparameters are passed to a trial in the Create operation.
# In this example, the model (defined in experiment_files/model_def.py) is expecting
# the following hyperparameters:
#   -> global_batch_size,
#   -> n_filters1
#   -> n_filters2,
#   -> learning_rate,
#   -> dropout,
#   -> dropout2.
def sample_params() -> Dict[str, object]:
    hparams = {
        "global_batch_size": 64,
        "n_filters1": random.randint(8, 64),
        "n_filters2": random.randint(8, 72),
        "learning_rate": random.uniform(0.0001, 1.0),
        "dropout1": random.uniform(0.2, 0.8),
        "dropout2": random.uniform(0.2, 0.8),
    }
    return hparams
