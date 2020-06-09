"""
It seems the hardest part of supporting the API I proposed last week was dealing with amp models.
This proposal extends the previously proposed TrialContext API with an amp_init() module:

    model = context.model(user_created_model())
    opt = context.optimizer(user_created_opt(model))

    # This is not ideal, but I think its not the end of the world.
    model, opt = context.amp_init(model, opt)

I found out that pytorch 1.6 will have its own amp module, and it's API is quite a bit cleaner than
the apex.amp module (and it seems to use no gloabls :)

It turns out that the pytorch-native amp API is pretty close to the API I proposed already, so I
made some minor adjustemnts to make sure that they were even closer (and that we could easily
support pytorch-native AMP in the future).

The question I still have is: can we automatically handle optimizer.zero_grad()?  Is that too
strong of an assumption?
"""

############

"""
Example 1: pytorch-native gradient scaling.

Notice that there is only one GradScaler in the training loop, just like there is only one
TrialContext object in a Determined PyTorchTrial.

code from https://pytorch.org/docs/master/notes/amp_examples.html#typical-mixed-precision-training
"""

# Creates model and optimizer in default precision
model = Net().cuda()
optimizer = optim.SGD(model.parameters(), ...)

# Creates a GradScaler once at the beginning of training.
scaler = GradScaler()

for epoch in epochs:
    for input, target in data:
        optimizer.zero_grad()

        # Runs the forward pass with autocasting.
        with autocast():
            output = model(input)
            loss = loss_fn(output, target)

        # Scales loss.  Calls backward() on scaled loss to create scaled gradients.
        # Backward passes under autocast are not recommended.
        # Backward ops run in the same precision that autocast used for corresponding forward ops.
        scaler.scale(loss).backward()

        # scaler.step() first unscales the gradients of the optimizer's assigned params.
        # If these gradients do not contain infs or NaNs, optimizer.step() is then called,
        # otherwise, optimizer.step() is skipped.
        scaler.step(optimizer)

        # Updates the scale for next iteration.
        scaler.update()


# The same thing in Determined:

class MyPytorchTrial(PyTorchTrial):
    def __init__(self, context):
        # There's one context, just like there was one GradScaler()
        # our context object will mimic the API of the GradScaler.
        self.context = context

        self.scaler = self.context.set_scaler(GradScaler())

        self.model = self.context.model(my_make_model())

        self.optimizer = self.context.optimizer(my_make_optimizer(self.model.parameters()))

    def train_batch(self, batch):
        input, target = batch

        # I propose we handle zero_grad() automatically... is there a downside to this?
        # self.optimizer.zero_grad()

        with autocast():
            output = self.model(input)
            loss = loss_fn(output, target)

        self.context.scale(loss).backward()

        # Since we are going to support gradient aggregation internally, we might ignore this call.
        self.context.step(self.optimizer)

        # No need for the user to call scaler.update(); we'll handle that for them.


"""
Example 2: multiple objects with new pytorch API

https://pytorch.org/docs/master/notes/amp_examples.html#working-with-multiple-models-losses-and-optimizers
"""

scaler = torch.cuda.amp.GradScaler()

for epoch in epochs:
    for input, target in data:
        optimizer0.zero_grad()
        optimizer1.zero_grad()
        with autocast():
            output0 = model0(input)
            output1 = model1(input)
            loss0 = loss_fn(2 * output0 + 3 * output1, target)
            loss1 = loss_fn(3 * output0 - 5 * output1, target)

        scaler.scale(loss0).backward(retain_graph=True)
        scaler.scale(loss1).backward()

        # You can choose which optimizers receive explicit unscaling, if you
        # want to inspect or modify the gradients of the params they own.
        scaler.unscale_(optimizer0)

        scaler.step(optimizer0)
        scaler.step(optimizer1)

        scaler.update()


# The same thing in Determined:

class MyPytorchTrial(PyTorchTrial):
    def __init__(self, context):
        self.context = context
        self.scaler = self.context.set_scaler(GradScaler())
        self.model0 = self.context.model(my_make_model())
        self.model1 = self.context.model(my_make_model())
        self.optimizer0 = self.context.optimizer(my_make_optimizer(self.model0.parameters()))
        self.optimizer1 = self.context.optimizer(my_make_optimizer(self.model1.parameters()))

    def train_batch(self, batch):
        input, target = batch

        # We deal with zero_grad() (do we?)

        with autocast():
            output0 = self.model0(input)
            output1 = self.model1(input)
            loss0 = loss_fn(2 * output0 + 3 * output1, target)
            loss1 = loss_fn(3 * output0 - 5 * output1, target)

        self.scaler.scale(loss0).backward(retain_graph=True)
        self.scaler.scale(loss1).backward()

        # Users clip gradients in callbacks specified in the scaler.step() operation
        # (this is because we handle gradient aggregation ourselves)
        def my_clip_grads_0(...):
            return ...

        def my_clip_grads_1(...):
            return ...

        self.scaler.step(self.optimizer0, clip_grads=my_clip_grads_0)
        self.scaler.step(self.optimizer1, clip_grads=my_clip_grads_1)

        # We handle scaler.update()


"""
Example 3: a proposal using the existing apex.amp mixed precision.

we should be basically API compatible with the GradScaler, even if we are not using it.  That way
when we do want to support it, it's easy, and also we are not making up an API totally from-scratch.
"""


class MyPytorchTrial(PyTorchTrial):
    def __init__(self, context):
        self.context = context
        mod0 = self.context.model(my_make_model())
        mod1 = self.context.model(my_make_model())
        opt0 = self.context.optimizer(my_make_optimizer(mod0.parameters()))
        opt1 = self.context.optimizer(my_make_optimizer(mod1.parameters()))

        # This should work fine with apex.amp, and has the added benefit that you no longer have
        # to configure amp via the experiment config... which in turn means you could do a
        # hyperparameter search over it, if you wanted to.
        (self.model0, self.model1), (self.optimizer0, self.optimizer1) = \
            self.context.amp_init([mod0, mod1], [opt0, opt1]

        # suppose the user wants one lr_scheduler auto-stepped every epoch...
        self.lr_scheduler0 = context.lr_scheduler(
            my_make_lr_scheduler(self.optimizer0), step_mode=STEP_EVERY_EPOCH
        )

        # and one scheduler stepped totally manually
        self.lr_scheduler1 = context.lr_scheduler(
            my_make_lr_scheduler(self.optimizer0), step_mode=MANUAL_STEP
        )

    def train_batch(self, batch):
        input, target = batch

        output0 = self.model0(input)
        output1 = self.model1(input)
        loss0 = loss_fn(2 * output0 + 3 * output1, target)
        loss1 = loss_fn(3 * output0 - 5 * output1, target)

        # TODO: name this better, but it is analagous to the scaler.scale() function
        self.context.loss_tensor(loss0).backward(retain_graph=True)
        self.context.loss_tensor(loss1).backward(retain_graph=True)

        def my_clip_grads_0(...):
            return ...

        def my_clip_grads_1(...):
            return ...

        # This is analgous to the scaler.step() function
        self.step(self.optimizer0, clip_grads=my_clip_grads_0)
        self.step(self.optimizer1, clip_grads=my_clip_grads_1)

        # The user opted to step lr_scheduler1 manually
        if decide_to_step_lr_scheduler():
            self.lr_scheduler1.step()
