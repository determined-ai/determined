# PTL Adapter

user flow:

- have a ptl project with lightningModule (LM), lightningTrainer, and maybe lightningDataModule (DM)
- have LM extend DETLM instead
  - implement (and/or pick) loss_fn
- have DM extend DETDM instead
  - impl methods to return detDataloader


## QUESTION

- What are dp, ddp2: data paraller and dist data prallel
  - do we support these with Horovad?
  - ddp seems to differ from dp and ddp2 (no batch parts)
  - https://pytorch-lightning.readthedocs.io/en/latest/multi_gpu.html#multi-gpu
- What is AMP

## LightningModule

LM wraps and organizes PyTorch code. [ref](https://pytorch-lightning.readthedocs.io/en/latest/lightning_module.html)

### TODO

check rest of lightning module hooks and methods
check the signatures

### API

methods:
- `configure_optimizers`: required; 
  - only a single optimizer case is supported
  - QUESTION: can return combination of optimizers and lr schedulers. what do we support here.
  - commit 183601649d1da6e5d941d0192718f3848f9e5625 multiple optimizer support for pytorch

- `forward`: required; inference only
- `freeze`: NUD; freeze  all params for inference
- `log`: NUD; can be called to log on_step or on_epoch with reduce_fx
- `log_dict`: NUD; can be called to log on_step or on_epoch with reduce_fx
- `print`: NUD;
- `save_hyperparameters`: NUD; saves model inputs in model.hparams. eg determined context?
  - QUESTION: do we have this work with det context?
- `test_step`: unsupported in determined
- `test_step_end`: unsupported in determined
- `test_epoch_end`: unsupported in determined
- `to_onnx`: NUD; given a file_path save in onxx format
  - QUESTION: support? in checkpoint?
- `to_torchscript`: NUD; export as torchscript
- `training_step`: required; full training loop
  - QUESTION hiddens and optimizer index arguments
- `training_step_end`
  - used for softmax or NCE loss
  - in dp or ddp2: this will be called with `[training_step(part) for part in batch_parts]`
  - QUESTION do we support dp, ddp2
- `training_epoch_end`: use this in case you need to do something with all the outputs for every training_step.
  - QUESTION: do we have such a hook in PyTorch trial
- `unfreeze`: NUD; unfreeze all params
- `validation_step`
  - TODO need batch index.
  - QUESTION: would we support multiple validation dataloaders? this fn would expect dataloader_idx if so
- `validation_step_end`
  - similar to other x_step_end
- `validation_epoch_end`
  - similar to other x_epoch_end

properties:
QUESTION: where does each of these get set. if it's the trainer how do we set it without it.
- `current_epoch`
- `device`: use to make your code device agnostic
- `global_rank`
- `global_step`
- `hparams`: QUESTION map to determined hparams or mark unsupported?
- `logger`
- `local_rank`
- `precision`
- `trainer`: pointer to the trainer
  - QUESTION unsupported?
- `use_amp`
- `use_ddp`
- `use_ddp2`
- `use_dp`
- `use_tpu`

hooks:
- `backward`: QUESTION how does it related to `manual_backward`
- `get_progress_bar_dict`: QUESTION do we support the default progress bar? if so we need this as well
- `manual_backward`: TODO support
- `on_after_backward`: TODO support
- `on_before_zero_grad`: QUESTION support? what's our pytorch support
- `on_fit_start`: QUESTION support?
- `on_fit_end`QUESTION support?
- `on_load_checkpoint`: QUESTION do we support this? how is our checkpoint generated.
- `on_save_checkpoint`: QUESTION do we support this?
- `on_pretrain_routine_start`: TODO what's the pretrain routine
- `on_pretrain_routine_end`: same
- `on_test_batch_start`: not supported
- `on_test_batch_end`: not supported
- `on_test_epoch_start`: not supported
- `on_test_epoch_end`: not supported
- `on_train_batch_start`: TODO support
- `on_train_batch_end`: TODO support
- `on_train_epoch_start`: QUESTION epoch hooks for PytorchTrial
- `on_train_epoch_end`: QUESTION epoch hooks for PytorchTrial
- `on_validation_batch_start`: TODO support
- `on_validation_batch_end`: TODO support
- `on_validation_epoch_start`: QUESTION epoch hooks for PytorchTrial
- `on_validation_epoch_end`: QUESTION epoch hooks for PytorchTrial
- `optimizer_step`: to control how often optimizers step. to adjust the default way the Trainer calls each optimizer.
  - affects train_batch
  - TODO support. seems like we need to support
- `optimizer_zero_grad`
- `prepare_data`: like datamodule
- `setup`: like teardown but on the begining. runs on every process in ddp
- `tbptt_split_batch`: When using truncated backpropagation through time. Huh?
  - QUESTION do we support?
- `teardown`: runs after fit and test stages
  - QUESTION corresponding hook
- `train_dataloader` alternative to datamodule
  - TODO support
- `val_dataloader`: alternative to datamodule
  - TODO support
- `test_dataloader`: not supported
- `transfer_batch_to_device` added if dataloader returns custom data structure
  - QUESTION do we support?

## DataModule

DM wraps and organizes PyTorch DataLoaders. [ref](https://pytorch-lightning.readthedocs.io/en/latest/lightning_module.html)
[ref](https://pytorch-lightning.readthedocs.io/en/latest/datamodules.html)


### TODO

- turn pytorch dataloader into determined pytorch dataloader
  - or ask user to provide conversions for their DataLoaders to DetDataLoader

### Unsupported?

- `def transfer_batch_to_device(self, batch, device):`
  - does `LightningDataModule` reference this internally?
