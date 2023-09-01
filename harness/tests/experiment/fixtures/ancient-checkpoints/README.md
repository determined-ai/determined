# Ancient Checkpoints

The checkpoint loading part of the system is different than most parts of the
system because it specifically needs to be built to handle the outputs of older
versions of the system.

We have some old checkpoints laying around here to ensure that we keep working
with old checkpoints in the future.

- `0.12.3-keras`: a stripped-down checkpoint, only useful for warmstarting
- `0.13.7-keras`: a stripped-down checkpoint, only useful for warmstarting
- `0.13.8-keras`: a stripped-down checkpoint, only useful for warmstarting
- `0.13.13-pytorch-flex`: fetched by Checkpoint.download() to populate metadata.json
- `0.13.13-pytorch-old`: fetched by Checkpoint.download() to populate metadata.json
- `0.17.6-keras`: fetched by Checkpoint.download() to populate metadata.json
- `0.17.6-pytorch`: fetched by Checkpoint.download() to populate metadata.json
- `0.17.7-keras`: fetch by direct access to checkpoint files
- `0.17.7-pytorch`: fetch by direct access to checkpoint files
- `0.20.0-pytorch`: a checkpoint from just before the PyTorchTrial Trainer API
- `0.21.0-pytorch`: a Trainer checkpoint with auto-importable Trial with kwargs
- `0.21.0-pytorch-main`: a Trainer checkpoint with Trial in `__main__` and kwargs
