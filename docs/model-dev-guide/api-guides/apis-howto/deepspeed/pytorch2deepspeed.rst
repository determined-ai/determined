.. _pytorch-to-deepspeed:

########################################
 ``PyTorchTrial`` to ``DeepSpeedTrial``
########################################

.. meta::
   :description: Learn how to adapt an existing PyTorchTrial to use DeepSpeed. This article explains how adapting an existing PyTorchTrial to use DeepSpeed mirrors the process for adapting existing code to use DeepSpeed outside of Determined.

Adapting an existing :class:`~determined.pytorch.PyTorchTrial` to use DeepSpeed mirrors the process
for adapting existing code to use DeepSpeed outside of Determined.

The first step is to switch to the DeepSpeed trial and context objects. Next, you need to initialize
the model engine and replace the context calls with appropriate replacements. Remember to modify the
experiment configuration, specifying an appropriate DeepSpeed configuration.

Reference conversion example:

.. code:: diff

   -class MyTrial(PyTorchTrial):
   +class MyTrial(DeepSpeedTrial):
        def __init__(self, context):
           self.context = context
           self.args = AttrDict(self.context.get_hparams())
           net = ...
           optimizer = ...
   -       self.model = self.context.wrap_model(net)
   -       self.optimizer = self.context.wrap_optimizer(optimizer)
   +       model_engine = deepspeed.initialize(
   +           args=self.args,
   +           model=net,
   +           optimizer=optimizer,
   +           ...
   +       )
   +       self.model = self.context.wrap_model_engine(model_engine)

       def build_training_data_loader(self) -> Any:
           trainset = ...
           return DataLoader(
               trainset,
   -           batch_size=self.context.get_per_slot_batch_size(),
   +           batch_size=self.model.train_micro_batch_size_per_gpu(),
               shuffle=True
           )

       def build_validation_data_loader(self) -> Any:
           valset = ...
           return DataLoader(
               valset,
   -           batch_size=self.context.get_per_slot_batch_size(),
   +           batch_size=self.model.train_micro_batch_size_per_gpu(),
               shuffle=True
           )

   -    def train_batch(self, batch, epoch_idx, batch_idx):
   +    def train_batch(self, iter_dataloader, epoch_idx, batch_idx):
   -       inputs, targets = batch
   +       inputs, targets = self.context.to_device(
   +           next(iter_dataloader)
   +       ) # Get a batch from the iterator
           outputs = self.model(inputs)
           loss = self.criterion(outputs, targets)
   -       self.context.backward(loss)
   -       self.context.step_optimizer(self.optimizer)
   +       self.model.backward(loss)
   +       self.model.step()
           return {"loss": loss}

   -    def evaluate_batch(self, batch, batch_idx):
   +    def evaluate_batch(self, iter_dataloader, batch_idx):
   -       inputs, targets = batch
   +       inputs, targets = self.context.to_device(
   +           next(iter_dataloader)
   +       ) # Get a batch from the iterator
           outputs = self.model(inputs)
           metric = ...
           return {"metric": metric}
