# The Determined Guide to Developing PyTorchTrial

"Most of PyTorchTrial is simple.  But the Horovod / AMP / Gradient Aggregation
combo is not."


## Basics of PyTorch

### The forward pass

Calculate an error in an autodifferentiable way.  Each step in the calculation
leaves some traceable path of autodiff magic on the tensors which you pass
through.

### The backward pass

Once you have arrived at the loss you want to minimize, follow the magic
traceable autodiff path backward to calculate the gradient.  The gradients are
left in-place on the tensors to which they apply.

### `optimizer.step()`

Gather up all of the gradents for the tensors this optimizer is responsible for
tuning, and apply the `gradient * learning_rate` to each weight.  Now you have
new weights, congratulations!


## Horovod optimizer

The horovod optimizer calculates gradient updates on each worker, then
aggregates the updates across each worker before updating the weights in the
model on each worker.

### `loss.backward()`

Horovod registers backward hooks on all of the `hvd_optimizer`'s parameter
tensors; each backward pass starts an asynchronous allreduce of the gradient
updates for parameters from that loss tensor.  The use of backward hooks makes
this API-transparent to users.

### `hvd_optimizer.synchronize()`

Gather the results of the asynchronous allreduce of gradient updates between
workers (don't apply any gradient updates).  This is useful if you need to
process the gradients before applying them to the model weights, such as for
gradient clipping or gradient scaling.

### `hvd_optimizer.step()`

First call .synchronize(), then apply the allreduced gradient updates by
calling .step() on the wrapped optimizer.  This is the most vanilla way to use
horovod.

### `with hvd_optimizer.skip_synchronize():`

Any call to `hvd_optimizer.step()` within this context manager will skip the
.synchronize() step, which is useful if you already called
`hvd_optimizer.synchronize()`.

### `hvd_optimizer.backward_passes_per_step`

Modifies the backward hooks to only start the asynchronous allreduce every
`backward_passes_per_step` steps.  This value is useful for either gradient
aggregation (multiple batches per optimizer.step()) or if you plan on calling
loss.backward() multiple times.

Note that in Determined, the option named `backward_passes_per_step` is only
for calling loss.backward() multiple times, and we multiply that value by the
gradient aggregation frequency to get the `backward_passes_per_step` that we
actually pass to Horovod.

### optimizer.compression=hvd.Compression.fp16

This is not AMP, and it does not behave like the GradScaler.  It just blindly
casts gradient update tensors to fp16 for network communication and blindly
casts them back to fp32 afterwards.


## Gradient Aggregation:

Gradient aggregation by a factor of N is just like multiplying the batch size
by N, except it saves on GPU memory because you process two batches and average
their gradients before you apply the gradients.  This makes sense in
distributed training because of the high communication cost of the allreduce.
(There are some cases where it's not "just like multiplying batch size by N",
such as with batch norm).

The only operation required for gradient aggregation are:

 * do N forward/backward passes before doing anything with the gradients that
   result from those backward passes.  Those in-place gradients will add
   together naturally.

 * Divide the gradients by N to match the behavior of an N-times-larger batch
   size.

 * Call optimizer.step() after every N batches to act on the aggregated
   gradients.


## PyTorch-Native AMP:

"AMP" stands for "automatic mixed precision", where certain calculations happen
in half-precision math (fp16).  This is normally only meaningful with GPUs
enabled, because Nvidia GPUs have special vector instructions for making fp16
math go really freaking fast.

### `with autocast():`

Choose certain parts of a model to be executed with fp16 math, based on some
rules that don't concern us here.  But with autocast(), you'll need to have a
GradScaler for stable training.

### The `GradScaler`

Multiply a loss by some factor before the backward pass, so that fp16 gradient
calculations don't overflow or underflow.  This scale factor is automatically
tuned in a pretty dumb control loop to keep it in a useful region.

### scaler.scale(loss):

Multiply the loss by the current scale factor.

### scaler.step(optimizer):

Check if the gradients which resulted caused an overflow/underflow.  If not,
step the optimizer.

### scaler.update():

Call the dumb control loop: if there was an overflow, decrease the scale
factor.  If there was an underflow, increase it.

Note that the `GradScaler` is designed to be one-per-training loop.  If there
are multiple optimizers involved, there should only be one `scaler.update()`
call after all `scaler.step(opt)` calls are made for a given batch.

## Nvidia apex AMP

### `model, optimizer = amp.initialize(model, optimizer)`

This is kind of like calling `with autocast()` for the forward pass or
something.  I'm guessing the reason that the optimizer is included is so that
the calls to optimizer.step() with underflown/overflown gradients can be safely
skipped, and at some point they must also trigger the scale factor control
loop.

### `with amp.scale_loss(loss, optimizer):`

Example:

```python
with amp.scale_loss(loss, optimizer) as scaled_loss:
    scaled_loss.backward()
```

This context manager is going to scale the loss by the current scale factor,
then unscale after the context manager exits.  Gradients are only accumulated
into tensors that this optimizer is responsible for.

Normally, apex.amp is fully capable of both gradient aggregation and dtrain.
That might look like this:

```python
# N-1 passes: don't unscale yet
with amp.scale_loss(loss, optimizer, defer_unscale=True) as scaled_loss:
    scaled_loss.backward()

# Nth pass: unscale now
with amp.scale_loss(loss, optimizer, defer_unscale=False) as scaled_loss:
    scaled_loss.backward()
    # synchronize before unscaling
    optimizer.synchronize()
```

However, our PyTorchTrial API doesn't allow that, because when
`context.backward(loss)` is called, we don't actually know what optimizer the
user intends to associate with that loss.

We definitely want to support apex.amp with dtrain, so we actually just "guess"
the optimizer by synchronizing all of the optimizers.  To ensure this "guess"
is correct, we only allow one optimizer with apex.amp.

This decision was made since long-term we expect the PyTorch-native API to
dominate and it is a much more flexible API which does not have the same API
limitations (needing to know the optimizer when calling `scale_loss`).

## The Laws of PyTorchTrial

* Horovod optimizer:
  * call .synchronize() zero or one times per optimizer.step().  If you do call
    it, wrap the .step() in a `with optimizer.skip_synchronize()` call.
  * always synchronize before gradient clipping; otherwise each worker clips
    differently
  * always synchronize before scaler.step(optimizer); otherwise each worker
    steps differently
  * you could scaler.scale(loss) before or after optimizer.synchronize(), that
    doesn't matter

* Gradient aggregation (by a factor of N):
  * do a forward and backward pass on each batch
  * average the summed gradients by dividing them by N (or equivalently, divide
    the loss by N before each backward pass)
  * set `hvd_optimizer.backward_passes_per_step` to N (aggregation frequency)
  * only synchronize/step optimizer every N batches
  * always call scaler.scale(loss) with the same scale factor for each batch in
    a grouping of N batches (only call scaler.update() every N batches); this
    ensures that the gradient aggregation is valid.

* PyTorch-native AMP:
  * (repeated from above) always call scaler.step(optimizer) *after*
    optimzer.synchronize()
  * (repeated from above) only call scaler.update() after each grouping of N
    batches.

* Nvidia apex AMP:
  * with horovod, you must call optimizer.synchronize() *before* the
    `amp.scale_loss()` context manager exits, because the underflow/overflow
    checks happen in that context manager's exit; otherwise each worker would
    step differently.
  * since the `context.backward(loss)` API does not specify an optimizer, and
    because `amp.scale_loss()` requires us to take actions on the correct
    optimizer, we do not allow multiple optimizers; we have to know that the
    our one optimizer is the correct optimizer to pass to scale_loss().

* PyTorch Distributed:
  * PyTorch Distributed wraps models in a `DistributedDataParallel` wrapper,
  which must be done after initializing APEX AMP.
  * PyTorch DDP synchronizes losses during the backwards pass
  * Since gradients are automatically synchronized during each backwards pass,
    each N-1 step must be wrapped in a `no_sync` context manager to avoid
    inefficiencies of syncing gradients at every step during gradient
    aggregation
  * DDP broadcasts parameters on model instantiation with
    `DistributedDataParallel()`, during which the `state_dict` from rank 0 is
    broadcast to all other ranks in the group. This is done automatically,
    unlike horovod's explicit use of `broadcast_parameters`
