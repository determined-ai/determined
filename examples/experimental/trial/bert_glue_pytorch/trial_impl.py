import argparse
import pathlib
import determined as det
from determined import experimental

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--test", dest='test', action="store_true", help="Specifies test mode", default=False
    )
    args = parser.parse_args()

    config = {
        "description": "PyTorch Bert",
        "searcher": {
            "name": "single",
            "metric": "acc",
            "max_length": {
                "batches": 400,
            },
            "smaller_is_better": True,
        },
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
        "entrypoint": "model_def:BertPytorch"
    }

    experimental.create(
        test=args.test,
        context_dir=str(pathlib.Path.cwd()),
        config=config,
    )
