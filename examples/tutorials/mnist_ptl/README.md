# PTL Adapter

user flow:

- have a ptl project with lightningModule (LM), lightningTrainer, and maybe lightningDataModule (DM)
- have LM extend DETLM instead
  - implement (and/or pick) loss_fn
- have DM extend DETDM instead
  - impl methods to return detDataloader


## QUESTION

- What are dp, ddp2: data parallel and dist data prallel: don't directly support these but we do 
  - do we support these with Horovad?
  - ddp seems to differ from dp and ddp2 (no batch parts)
  - https://pytorch-lightning.readthedocs.io/en/latest/multi_gpu.html#multi-gpu
- What is AMP: automatic mixed precision

### Responses

- no dp ddp 2
- globalbatch = local batch * number of gpus
- for anything we don't support print warning. use our moneky patching library
  - monkey_patch.py

## LightningModule

LM wraps and organizes PyTorch code. [ref](https://pytorch-lightning.readthedocs.io/en/latest/lightning_module.html)

### TODO

- sort out what won't be in phase one or be unsupported
  - time estimate
- put out an initial pr in experimental with mnist

### API

#### Will Support:

methods:
- `configure_optimizers`: required; 
  - only a single optimizer case is supported
    - TODO figure out the other cases: support at least what's supported in pytorch trial.
  - commit 183601649d1da6e5d941d0192718f3848f9e5625 multiple optimizer support for pytorch
- `freeze`: NUD; freeze  all params for inference
  - r: no direct work
- `training_step_end`
  - used for softmax or NCE loss
  - in dp or ddp2: this will be called with `[training_step(part) for part in batch_parts]`
  - TODO checkout trainer sourcecode.
- `training_epoch_end`: use this in case you need to do something with all the outputs for every training_step.
  - no support in `pytorch/_callback`.
  - will depend on adding this hook to pytorch trial

- `forward`: required; inference only
- `save_hyperparameters`: NUD; saves model inputs in model.hparams. eg determined context?
  - no work needed? `get_hparam`. TODO check source code
- `training_step`: required; full training loop
  - hiddens: passed in for bptt (rnn)
    - this argument won't be supported TODO create pytorch trial ticket for supporting?
  - TODO optimizer index arguments
- `unfreeze`: NUD; unfreeze all params
  - r: no direct work. checkout trainer for freeze/unfreeze around trainstep
- `validation_step`
  - TODO need batch index.
  - if we support multiple validation dataloaders? this fn would expect dataloader_idx
    - out of scope for now. would require changing pytorchtrial api
- `validation_step_end`
  - similar to other x_step_end
- `validation_epoch_end`
  - similar to other x_epoch_end

properties:
TODO: find where does each of these gets set in pytorch trial or lower and update at the right time
- `precision`: type of precision used. AMP = read and set? use apex library. called in wrapoptimizers. users need to configure in a config
  - configure-apex-amp fn
  - TODO read how configure-apex is used and figure out how this merges into lightning module. no way to directly support configure-apex?
- `current_epoch`: figure out a good place to update this
- `device`: use to make your code device agnostic
- `global_rank`: from context.distributed
- `global_step`: from context.distributed
- `hparams`: leave as is. no work to do
- `local_rank`: rank on a machine context.distributed.local_rank
- `use_amp`: using
- `use_ddp`: False
- `use_ddp2`: False
- `use_dp`: False
- `use_tpu`: False

hooks:
TODO Need added support in our pytorch callback. construct a callback class with most of these hooks. only support relatively easy or nontrainer hooks

- `on_before_zero_grad`: TODO
  - need to change pytorch `step_optimizer` auto zero grad
- `on_after_backward`: TODO
- `on_fit_start`: not supported. a trainer fn
- `on_fit_end`: not supported. a trainer fn
- `on_load_checkpoint`: TODO
  - Gives model a chance to load something before state_dict is restored.
- `on_save_checkpoint`: TODO
- `on_train_batch_start`: TODO support
- `on_train_batch_end`: TODO support
- `on_train_epoch_start`:
  - needs callback support in pytorchcallback
- `on_train_epoch_end`:
  - needs callback support in pytorchcallback
- `on_validation_batch_start`: TODO support
- `on_validation_batch_end`: TODO support
- `on_validation_epoch_start`:
  - needs callback support in pytorchcallback
- `on_validation_epoch_end`:
  - needs callback support in pytorchcallback
- `prepare_data`: like datamodule: user can use in buildxdataloader
  - no direct work? we won't directly use in adapter. user can use as usual.
  - what about distributed setup. call during init?
- `train_dataloader` alternative to datamodule
  - no direct work? user will use it in buildxdataloader
- `val_dataloader`: alternative to datamodule
  - no direct work? user will use it in buildxdataloader

#### Not (Immediately) Supported

methods:
- `log`: NUD; can be called to log on_step or on_epoch with reduce_fx
  - r: exclude or print a warning. or have them simply print
- `log_dict`: NUD; can be called to log on_step or on_epoch with reduce_fx
  - r: exclude or print a warning. or have them simply print
- `print`: NUD;
    - r: exclude or print a warning. or have them simply print
- `test_step`: unsupported in determined
- `test_step_end`: unsupported in determined
- `test_epoch_end`: unsupported in determined
- `to_onnx`: NUD; given a file_path save in onnx format
  - needs support for saving arbitrary files in checkpoint.
  - potentially out of scope
- `to_torchscript`: NUD; export as torchscript
  - same as `to_onnx`

properties:
- `logger`: no need to support. but what do we do if user uses it.
  - fake logger?
- `trainer`: pointer to the trainer
  - unsupported = raise and warn?

hooks:
- `backward`:
  - instead of loss.backward user should call context.backward but with what api
  - more involved
  - potentially skip for first pass. 
- `manual_backward`: check with trainer. new hook.
  - this lets user call backward themselves
  - we can ensure that all the proper scaling when using 16-bit etc has been done for you: needs checking with trainer
  - good candidate for leaving for next milestones. skip for first pass. 
- `on_pretrain_routine_start`: internal to trainer. don't support
- `on_pretrain_routine_end`: internal to trainer. don't supportsame
- `on_test_batch_start`: not supported
- `on_test_batch_end`: not supported
- `on_test_epoch_start`: not supported
- `on_test_epoch_end`: not supported
- `optimizer_step`: to control how often optimizers step. to adjust the default way the Trainer calls each optimizer.
  - affects train_batch
  - an internal trainer hook?
  - potentially skip for first pass. user can use pytorchtrial or lightningtrial with trainer
  - fail if the user is defining this
- `optimizer_zero_grad`: control when zerograd is called
  - potentially skip for first pass. 
- `setup`: like teardown but on the beginning. runs on every process in ddp
  - either run at begining of training or skip since fit is trainer internal
  - not supported
- `get_progress_bar_dict`: modify progress bar display
  - this is a trainer feature
- `tbptt_split_batch`: when using truncated backpropagation through time: not supported
- `teardown`: runs after fit and test stages. no fit or test. not supported
- `test_dataloader`: not supported
- `transfer_batch_to_device` added if dataloader returns custom data structure
  - not supported in pytorch trial. skip

## DataModule

DM wraps and organizes PyTorch DataLoaders. [ref](https://pytorch-lightning.readthedocs.io/en/latest/lightning_module.html)
[ref](https://pytorch-lightning.readthedocs.io/en/latest/datamodules.html)

  - does `LightningDataModule` reference `transfer_batch_to` internally?

### TODO

- turn pytorch dataloader into determined pytorch dataloader
  - or ask user to provide conversions for their DataLoaders to DetDataLoader
