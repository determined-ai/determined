"""
High-level Sprinkle API for training with pytorch_lightning.
"""

# ExperimentConfig settings:
#
#   For Horovod training:
#
#       #launch_layer: python3 -m determined.launch.auto_horovod
#       entrypoint_script: python3 train_lightning.py
#
#   For DistributedDataParallel (PTL reccommended backend):
#
#       launch_layer: null
#       entrypoint_script: python3 train_lightning.py
#

import pytorch_lightning as pl

import determined as det
import determined.pytorch_lightning as det_pl

import data
import model_def

context = det_pl.init()

trainer = context.Trainer(
    # most arguments are passed directly
    ...
    # special arguments that we normally handle automatically
    accelerator=...
    resume_from_checkpoint=...
    gpus=...
    num_nodes=...
    num_processes=...
    # disallowed arguments
    max_steps=...
    max_epochs=...
    min_epochs=...
    min_steps=...
)

trainer.fit(...)



"""
High-level Sprinkle API for DBI pytorch_lightning.
"""

context = det_pl.init()

ckpt = Determined().get_checkpoint(...)
resume_path = ckpt.load()

trainer = context.Trainer(
    ...
    resume_from_checkpoint = resume_path,
)

my_model = ...
my_dataloaders = ...
my_datamodule = ...

predictions = trainer.predict(my_model, my_dataloaders, my datamodule)
