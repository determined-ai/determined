import argparse
import pathlib
import determined as det

import model_def

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--mode", dest="mode", help="Specifies test mode or submit mode.", default="submit"
    )
    args = parser.parse_args()

    config = {
        "description": "PyTorch Bert",
        "searcher": {"name": "single", "metric": "acc", "max_steps": 4, "smaller_is_better": True},
        "data": {
            "data_dir": "/tmp/data",
            "task": "MRPC",
            "model_name_or_path": "bert-base-uncased",
            "output_mode": "classification",
            "path_to_mrpc": "",
            "download_data": True,
        },
        "hyperparameters": {
            "global_batch_size": det.Constant(value=24),
            "model_type": det.Constant(value="bert"),
            "learning_rate": det.Constant(value=0.00002),
            "lr_scheduler_epoch_freq": det.Constant(value=1),
            "adam_epsilon": det.Constant(value=1e-8),
            "weight_decay": det.Constant(value=0),
            "num_warmup_steps": det.Constant(value=0),
            "num_training_steps": det.Constant(value=459),
            "max_seq_length": det.Constant(value=128),
        },
    }

    det.create(
        trial_def=model_def.BertPytorch,
        mode=det.Mode(args.mode),
        context_dir=str(pathlib.Path.cwd()),
        config=config,
    )
