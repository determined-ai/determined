"""
I was thinking about how we might support multiple models/optimizers/schedulers with fewer
callbacks.  As great as pytorch-lightning is, I think their treatment of multiple optimizers is
pretty weird; you get a train_batch() call once for every optimzer?  Why do we even need to track
the optimizers that closely?

All we need at the end of the day is to hook into specific behaviors.  In a single-model,
single-optimizer, single-scheduler case, it's pretty easy for us to have a callback for building
each thing and to carefully keep track of each thing.  But that makes less sense when you have
arbitrary numbers of each thing in the mix.

So this is a proposal to support multiplemodels/optimizers/schedulers in a way that feels more
pytorch-native, where the burden of writing callbacks is on us rather than on the user.

An obvious drawback is that it is easier for the user to forget to call context.backward(loss)
instead of loss.backward() for instance, but I think that is counter-balanced by having an API
that is just plain easier for users to wrap their heads around.


Compared to the proposed API of Pytorch Flexible Primitives, I don't see any ways that this API is
any more or less capable; that API is definitely not bad.  I just had some ideas of how we support
the same features in a way that was hopefully easier for users, and I wanted to post this to see
what other people thought.
"""

class PyTorchTrialContext(det.TrialContext):
    """
    I added some new functions to the PyTorchContext, which we need to help users get the proper
    Determined functionality out of PyTorch things.
    """

    def __init__(...):
        # map lr_schedulers to lr_helpers
        self._lr_helpers = {}

        # We collect saveable things as calls are made to the context
        # (Of course, we could still have callbacks for saving/loading arbitrary things)
        self._models_to_save = []
        self._optimizers_to_save = []
        self._lr_schedulers_to_save = []

    def model(module):
        """
        I chose to make this a function instead of
        """
        model = model.to(self._device)
        if self._n_gpus > 1:
            model = nn.DataParallel(self.context.model)
        # remember to save this model
        self._models_to_save.append(model)
        return model

    def optimizer(opt):
        if self._hvd_config.use:
            use_compression = self.hvd_config.fp16_compression
            opt = hvd.DistributedOptimizer(
                self.context.optimizer,
                named_parameters=self.context.model.named_parameters(),
                backward_passes_per_step=self.hvd_config.aggregation_frequency,
                compression=hvd.Compression.fp16 if use_compression else hvd.Compression.none,
            )
        # remember to save this optimzer
        self._optimizers_to_save.append(opt)
        return opt

    # TODO: this is a redundant specification of optimizer here:
    def lr_scheduler(lr_sched, opt):
        # save this helper for this scheduler
        self._lr_helpers[opt] = LRHelper(lr_sched)
        # remember to save this lr_scheduler
        self._lr_schedulers_to_save.append(lr_sched)
        return lr_sched

    def backward(loss):
        """
        If context.module() returned a custom nn.Module then this function could be eliminated
        from the user-facing API; it would be registered as a backwards hook of the torch.Tensor
        returned by our custom nn.Module.forward().
        """
        if self._use_amp:
            with apex.amp.scale_loss(loss, self.context.optimizer) as scaled_loss:
                return scaled_loss.backward()
        else:
            return loss.backward()

    def step_optimizer(opt):
        """
        If context.optimizer() returned a custom subclass, then this function could be eliminated,
        as we could just hook into the opt.step() call.
        """
        communicate_and_update = (batch_idx + 1) % self._hvd_config.aggregation_frequency == 0
        if communicate_and_update:
            if self._hvd_config.use:
                opt.synchronize()

            parameters = (
                # TODO: self.context.model.parameters() won't work here, but it isn't perfect on
                #       tip-of-master either:
                self.context.model.parameters()
                if not self.use_amp()
                else apex.amp.master_params(opt)
            )

            if self._hvd_config.average_aggregated_gradients:
                self._average_gradients(
                    parameters=parameters, divisor=self._hvd_config.aggregation_frequency
                )

            self._clip_grads(parameters)

            if self._hvd_config.use:
                with opt.skip_synchronize():
                    opt.step()
            else:
                opt.step()
            opt.zero_grad()

    def step_lr_scheduler(lr_sched):
        """
        If context.lr_scheduler() returned a custom subclass, then this function could be
        eliminated.  I seem to remember we had some trouble with doing it that way before though.
        """
        # retreive the helper for this scheduler
        lr_helper = self._lr_helpers[lr_sched]

        if self.lr_helper.should_step_lr(
            batches_completed=batch_idx + 1,
            epoch_length=len(self.training_loader),
            aggregation_frequency=self._hvd_config.aggregation_frequency,
        ):
            self.lr_helper.step()


############# User code example:


class MyPyTorchTrial():
    def __init__(self, context):
        self.context = context

        # wrap models
        self.model_a = self.context.model(make_model_a(...))
        self.model_b = self.context.model(make_model_b(...))
        # wrap optimizers
        self.opt_a = self.context.optimizer(make_opt(...))
        self.opt_b = self.context.optimizer(make_opt(...))

    def train_batch(self, batch, epoch_idx, batch_idx):
        images, targets = batch
        if epoch_idx % 1:
            # odd epochs for one model
            loss = self.model_a(images, targets)
            self.context.backward(loss)
            self.context.step_optimizer(self.opt_a)

        else:
            # even epochs on the other model
            loss = self.model_b(images, targets)
            self.context.backward(loss)
            self.context.step_optimizer(self.opt_b)

        # user now has full control over the shape of their metrics
        return {"loss": loss}

    def evaluate_batch(self, batch):
        images, targets = batch
        loss_a = self.model_a(image, targets)
        loss_b = self.model_b(image, targets)

        # just average losses or whatever
        return {"avg_loss": torch.mean(loss_a, loss_b)}

# Advantages of this proposed API:
#   - they call our functions rather much more often than they give us callbacks that we call,
#     which makes porting models easier (less refactoring)
#   - no weird morphing function prototypes; pytorch-lightning has train_batch(*args, **kwargs)
#     since the prototype depends on like three or four different settings
#   - we make fewer assumptions about how users want to write their code
#   - there's no difference in how users use the API in either the multi- or single-model cases

# Drawbacks that I am aware of:
#   - users could possibly forget to call context.(thing) for one of their things
