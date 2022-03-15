# Using DeepSpeed with Determined
DeepSpeed is a library released by Microsoft that supports large-scale distributed learning with
sharded optimizer state training and pipeline parallelism.  Using DeepSpeed with Determined is 
supported through our `DeepSpeedTrial` API.  

DeepSpeed Features:
* Zero Redundancy Optimizer (ZeRO) with parameter and optimizer state offloading
* Pipeline parallelism with interleaved microbatch training

Known Limitations of DeepSpeed:
* The primary limitation to be aware of is that pipeline parallelism can only be combined with ZeRO stage 1.  
* Parameter offloading is only supported with ZeRO stage 3.
* Optimizer offloading is only supported with ZeRO stage 2 and 3.

## Basics of DeepSpeed

### Configuration
DeepSpeed is usually used with a configuration file specifying the settings for various DeepSpeed features.
In lieu of Determined, this configuration file is passed when launching a training job:
```
deepspeed train.py --deepspeed_config=ds_config.json 
```
The configuration file path is parsed into an arguments object (e.g. `args`), which is used later to 
create the DeepSpeed model engine.

### Initialization
Initializing DeepSpeed training consists of two steps:
1. Initializing the distributed backend.
2. Creating the DeepSpeed model engine.

This is usually done with something like below:
```python
import deepspeed

...

deepspeed.init_distributed(dist_backend=args.backend) # backend usually nccl
net = ...
model_engine, optimizer, lr_scheduler, dataloader = deepspeed.initialize(
    args=args, # args has deepspeed_config as an attribute
    net=net,
    ...
)
```

### Training
Once the DeepSpeed model engine is created, the training process will depend on whether pipeline
parallelism is being used.

#### Data Parallel Training
For just data parallel training, the forward and backward steps are performed as follows:
```python
outputs = model_engine(inputs)
loss = criterion(outputs, targets)
model_engine.backward(loss)
model_engine.step()
```
Note how `backward` and `step` are called on the DeepSpeed model engine instead of `loss` and the optimizer respectively.

#### Pipeline Parallel Training
If a user wants to use pipeline parallelism, they will need to pass layers of their model to 
DeepSpeed's PipelineModule before creating the DeepSpeed model engine:
```python
net = ...
net = PipelineModule(
    layers=get_layers(net), # get_layers is a user provided function that will return layers of a network.
    loss_fn=torch.nn.CrossEntropyLoss(),
    num_stages=args.pipeline_parallel_size,
    ...
)
model_engine, _, _, _ = deepspeed.initialize(
    args=args,
    model=net,
    dataset=dataset, # optional
    ...
)
```
When using pipeline parallelism, DeepSpeed expects the configuration file to have `train_batch_size` and
`train_micro_batch_size_per_gpu` to be available so it can automatically interleave multiple microbatches
for processing in a single training schedule.  
If a `dataset` is passed to `deepspeed.initialize`, the model_engine will build an internal data loader that
creates batches of size `train_micro_batch_size_per_gpu`, which can be specified in the DeepSpeed config. 
You can also create your own dataloader and use that directly.
```python
model_engine.set_dataloader(train_dataloader)
for _ in range(train_iters):
    # The model_engine will automatically perform forward, backward, and optimizer update on 
    # batches requested internally from the dataloader and interleave.  
    model_engine.train_batch() 
```

### Putting it together

#### Data Parallel Training
For just data parallel training, the forward and backward steps are performed as follows:
```python
deepspeed.init_distributed(dist_backend=args.backend) # backend usually nccl
net = ...
model_engine, optimizer, lr_scheduler, dataloader = deepspeed.initialize(
args=args, # args has deepspeed_config as an attribute
net=net,
...
)
train_dataloader = ...
for idx, batch in enumerate(dataloader):
    inputs, targets = batch
    outputs = model_engine(inputs)
    loss = criterion(outputs, targets)
    model_engine.backward(loss)
    model_engine.step()
```

#### Pipeline Parallel Training
```python
deepspeed.init_distributed(dist_backend=args.backend) # backend usually nccl
net = ...
net = PipelineModule(
    layers=get_layers(net), # get_layers is a user provided function that will return layers of a network.
    loss_fn=torch.nn.CrossEntropyLoss(),
    num_stages=args.pipeline_parallel_size,
    ...
)
model_engine, _, _, _ = deepspeed.initialize(
    args=args,
    model=net,
    dataset=dataset, # optional
    ...
)
train_dataloader = ...
model_engine.set_dataloader(train_dataloader) # creates iterator over train_dataloader stored in model_engine
for _ in range(train_iters):
    # The model_engine will automatically perform forward, backward, and optimizer update on 
    # batches requested internally from the dataloader and interleave.  
    model_engine.train_batch()
```

## Using DeepSpeedTrial
You can think of `DeepSpeedTrial` as a way to use an automated training loop with DeepSpeed. Next, we'll demonstrate how the typical usage of DeepSpeed maps over to Determined.

### Determined's Experiment Configuration
Configuration Determined experiments for DeepSpeed is largely the same as doing so for PyTorchTrial 
with a few differences. 
* You will need to specify a required `hyperparameter` called `data_parallel_world_size` to explicitly
tell Determined how many model replicas you expect there to be.  
* You still need to provide `hyperparameters.global_batch_size` but you should make sure that this
matches `train_batch_size` in the DeepSpeed config.

You have control over how you pass a DeepSpeed configuration file for use to initialize the DeepSpeed model engine.  
One natural way is to specify it as a hyperparameter and treat the hyperparameters field as arguments
to pass to DeepSpeed.  

Your Determined experiment config might look something like this:
```yaml
hyperparameters:
  global_batch_size: 32 # this should match the train_batch_size you set in your DeepSpeed config
  data_parallel_world_size: 2
  deepspeed_config: base_ds_config.json
  ...
```
Then we can treat the hyperparameters section of the experiment config as the `args` that we pass
to `deepspeed.initialize` with `deepspeed_config` as a field.

### Implementing the Trial API
The example below shows the methods you need to implement for the `DeepSpeedTrial` API.  The interface
is largely the same as that for `PyTorchTrial` with the exception that `train_batch` and `evaluate_batch` 
take an iterator over a dataloader as input instead of a batch.  You can think of implementing
this interface as adding additional structure to the code above with the addition of a few additional
lines to support using DeepSpeed with Determined.

```diff
import deepspeed
from determined.pytorch import DataLoader, DeepSpeedTrial, DeepSpeedTrialContext

class MyTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
+       self.context = context
+       self.args = AttrDict(self.context.get_hparams()) # Get the hyperparameters from the experiment config
        net = ...
        model_engine, _, _, _ = deepspeed.initialize(
            args=self.args, # args has deepspeed_config as a field 
            model=net, 
            ...
        )
        # Register the model_engine with Determined so we can automatically support fault tolerance
        # among other things for you behind the scenes.
+       self.model_engine = self.context.wrap_model_engine(model_engine) 

    def build_training_data_loader(self) -> Any:
        trainset = ...
        return DataLoader(trainset, batch_size=self.context.get_micro_batch_size_per_gpu(), shuffle=True)

    def build_validation_data_loader(self) -> Any:
        valset = ...
        return DataLoader(valset, batch_size=self.context.get_micro_batch_size_per_gpu(), shuffle=False)

    def train_batch(
        self, iter_dataloader: Iterable[DataLoader], epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
+       inputs, targets = next(iter_dataloader) # Get a batch from the iterator
        outputs = self.model_engine(inputs)
        loss = self.criterion(outputs, targets)
        self.model_engine.backward(loss)
        self.model_engine.step()
        return {"loss": loss}

    def evaluate_batch(self, iter_dataloader: Iterable[DataLoader], batch_idx: int) -> Dict[str, Any]:
+       inputs, targets = next(iter_dataloader) # Get a batch from the iterator
        outputs = self.model_engine(inputs)
        metric = ...
        return {"metric": metric}
```

The process is very similar for pipeline parallel training but there won't be a need to manually
get a batch from the iterator.
```diff
import deepspeed
from determined.pytorch import DataLoader, DeepSpeedTrial, DeepSpeedTrialContext, DeepSpeedMPU

class MyTrial(DeepSpeedTrial):
    def __init__(self, context: DeepSpeedTrialContext) -> None:
+       self.context = context
+       self.args = AttrDict(self.context.get_hparams()) # Get the hyperparameters from the experiment config
        net = ...
        net = PipelineModule(
            layers=get_layers(net), # get_layers is a user provided function that will return layers of a network.
            loss_fn=torch.nn.CrossEntropyLoss(),
            num_stages=args.pipeline_parallel_size,
            ...
        )
        model_engine, _, _, _ = deepspeed.initialize(
            args=self.args, # args has deepspeed_config as a field 
            model=net, 
            ...
        )
        # Register the model_engine with Determined so we can automatically support fault tolerance
        # among other things for you behind the scenes.
+       self.model_engine = self.context.wrap_model_engine(model_engine)

    def build_training_data_loader(self) -> Any:
        trainset = ...
        return DataLoader(trainset, batch_size=self.context.get_micro_batch_size_per_gpu(), shuffle=True)

    def build_validation_data_loader(self) -> Any:
        valset = ...
        return DataLoader(valset, batch_size=self.context.get_micro_batch_size_per_gpu(), shuffle=False)

    def train_batch(
        self, iter_dataloader: Iterable[DataLoader], epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        loss = self.model_engine.train_batch(iter_dataloader)
        return {"loss": loss}

    def evaluate_batch(self, iter_dataloader: Iterable[DataLoader], batch_idx: int) -> Dict[str, Any]:
        loss = self.model_engine.eval_batch(iter_dataloader)
        return {"metric": metric}
```

### Switching from `PyTorchTrial` to `DeepSpeedTrial`
Adapting an existing `PyTorchTrial` to use DeepSpeed is pretty straightforward and closely mirrors
the process for adapting existing code to use DeepSpeed outside of Determined.  First step is to switch
over to the DeepSpeed trial and context objects.  Then, you'll need to initialize the model engine and 
replace the context calls with the appropriately replacements.  Remember to also modify the experiment config
to include the path to the DeepSpeed config in `hyperparameters.deepspeed_config`.
```diff
-class MyTrial(PyTorchTrial):
+class MyTrial(DeepSpeedTrial):
     def __init__(self, context):
        self.context = context
        self.args = AttrDict(self.context.get_hparams()) # Get the hyperparameters from the experiment config
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
        # The DeepSpeed model_engine object has the model, optimizer, and lr_scheduler (optional) as attributes
        # so we only need to register the model_engine with Determined.
+       self.model = self.context.wrap_model_engine(model_engine)

    def build_training_data_loader(self) -> Any:
        trainset = ...
-       return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size(), shuffle=True)
+       return DataLoader(trainset, batch_size=self.context.get_micro_batch_size_per_gpu(), shuffle=True)

    def build_validation_data_loader(self) -> Any:
        valset = ...
-       return DataLoader(valset, batch_size=self.context.get_per_slot_batch_size(), shuffle=False)
+       return DataLoader(valset, batch_size=self.context.get_micro_batch_size_per_gpu(), shuffle=False)

-    def train_batch(self, batch, epoch_idx, batch_idx):
+    def train_batch(self, iter_dataloader, epoch_idx, batch_idx):
-       inputs, targets = batch
+       inputs, targets = next(iter_dataloader) # Get a batch from the iterator
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
+       inputs, targets = next(iter_dataloader) # Get a batch from the iterator
        outputs = self.model(inputs)
        metric = ...
        return {"metric": metric}
```

#### A note about gradient accumulation/aggregation steps
The DeepSpeed config has a few fields that are used to determine how often the optimizer actually
takes a step using gradients to update the model weights.  The relevant fields are:

* `train_batch_size`: the total number of samples processed by all GPUs before an optimizer update
* `train_micro_batch_size_per_gpu`: the number of samples processed in each forward and backward pass by a single GPU
* `gradient_accumulation_steps`: the number of micro batches to process before taking a gradient step

The relationship between the fields is `train_batch_size` = `data_parallel_size` * `train_micro_batch_size_per_gpu` * `gradient_accumulation_steps`.
At least two of the three fields are required for a given DeepSpeed configuration.

The training dataloaders should always return batches of size `train_micro_batch_size_per_gpu`.  Determined will automatically
handle calling `train_batch` so that one effective batch in terms of metrics reporting is equal to processing `train_batch_size` across all GPUs.


### Advanced Usage: Custom Model Parallelism
For data parallel training with DeepSpeed, we will build the dataloader and average metrics on all GPU slots as we do normally.
If the model engine passed to `context.wrap__model_engine` is a `PipelineModule`, we will use the associated ModelParallelUnit (MPU) 
to decide on which slots to build the dataloader and average metrics.  If you want to change this behavior, you can
pass your own custom MPU unit to `context.wrap_mpu`.  The MPU object should have the following class methods implemented.

```python
class MyMPU(determined.pytorch.ModelParallelUnit):
    def get_global_rank(self) -> int:
        ...

    def get_data_parallel_rank(self) -> int:
        ...

    def get_data_parallel_world_size(self) -> int:
        ...

    def is_first_pipeline_stage(self) -> bool:
        ...

    def is_last_pipeline_stage(self) -> bool:
        ...

    def should_report_metrics(self) -> bool:
        ...
    
    def should_build_data_loader(self) -> bool:
        ...
```
