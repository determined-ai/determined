# PTL Adapter


## LightningModule

LM wraps and organizes PyTorch code. [ref](https://pytorch-lightning.readthedocs.io/en/latest/lightning_module.html)

### TODO

check rest of lightning module hooks and methods

methods: TODO check the signatures
- `configure_optimizers`
- `forward`
- `freeze`
- `log`
- `log_dict`
- `print`
- `save_hyperparameters`
- `test_step`
- `test_step_end`
- `test_epoch_end`
- `to_onnx`
- `to_torchscript`
- `training_step`
- `training_step_end`
- `training_epoch_end`
- `unfreeze`
- `validation_step`
- `validation_step_end`
- `validation_epoch_end`

hooks:
- `backward`
- `get_progress_bar_dict`
- `manual_backward`
- `on_after_backward`
- `on_before_zero_grad`
- `on_fit_start`
- `on_fit_end`
- `on_load_checkpoint`
- `on_save_checkpoint`
- `on_pretrain_routine_start`
- `on_pretrain_routine_end`
- `on_test_batch_start`
- `on_test_batch_end`
- `on_test_epoch_start`
- `on_test_epoch_end`
- `on_train_batch_start`
- `on_train_batch_end`
- `on_train_epoch_start`
- `on_train_epoch_end`
- `on_validation_batch_start`
- `on_validation_batch_end`
- `on_validation_epoch_start`
- `on_validation_epoch_end`
- `optimizer_step`
- `optimizer_zero_grad`
- `prepare_data`
- `setup`
- `tbptt_split_batch`
- `teardown`
- `train_dataloader`
- `val_dataloader`
- `test_dataloader`
- `transfer_batch_to_device`

## DataModule

LM wraps and organizes PyTorch DataLoaders. [ref](https://pytorch-lightning.readthedocs.io/en/latest/lightning_module.html)
[ref](https://pytorch-lightning.readthedocs.io/en/latest/datamodules.html)


### TODO

- turn pytorch dataloader into determined pytorch dataloader
  - or ask user to provide conversions for their DataLoaders to DetDataLoader

### Unsupported?

- `def transfer_batch_to_device(self, batch, device):`
  - does `LightningDataModule` reference this internally?
