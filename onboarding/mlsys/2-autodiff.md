# Part 2: Autodifferentiation and Optimizers

In "Part 1: Calculating Gradients and Averaging Them Together", we learned how
gradient descent works mathematically.

In this exercise, you'll learn how an actual deep learning library implements
gradient descent.  You'll also learn about what optimizers are in a deep
learning framework, and you'll implement one yourself, in both PyTorch and
Tensorflow.  And lastly, you'll

## Autodifferentiation

Autodifferentiation is a mechanism by which gradients may be calculated
symbolically for a model of arbitrary size and complexity by repeated
applications of the product rule.

You may remember from calculus class that the chain rule says that for
function $h$ composed of functions $f$ and $g$:

```math
h(x) = f(g(x))
```

then the derivative of $h$ is:

```math
h'(x) = f'(g(x)) \cdot g'(x)
```

The chain rule applies to derivatives of nested functions.  If we think of
successive layers of a model as nested functions, then we can imagine the
following:

- $x$ is the input to the model
- $g$ is the composition of all previous layers of the model
- $g(x)$ is the output of the previous layer
- $f$ is the current layer of the model
- $h$ is the composition of all previous layers and also the current one

Now let's replace the term $f'(g(x))$ with an arbitrary variable name $Z$:

```math
h'(x) = Z \cdot g'(x)
```

We can say that $Z$ must be a value that, multiplied by the gradient of previous
layers ($g'(x)$), results in the gradient including the current layer ($h'(x)$).

If $Z$ were particularly easy to calculate, it would be a way to calculate the
total gradient incrementally, by calculating the equivalent $Z$ term for each
layer and multiplying them together.  But is $Z$ easy to calculate for a layer?

The answer turns out to be "yes".  $Z$ is the result of feeding the output of
the previous layer $g(x)$, which is easy to compute, into $f'$, the derivative
of the operator $f$, which is often trivial to obtain (for example, if $f(n) =
n^2$ then $f'(n) = 2n$).

And so if we break a model into individual operations and track the output of
each layer, then we can express what contribution each layer makes to the total
gradient.

### A Complete Example, By Hand

Let's remember our loss function from the previous exercise:

```math
\mathrm{loss_point} = (m x - \mathrm{ytrue})^2
```

We can think of the output as a sequence of three operations applied to a
starting value $x$:

- multiply by $m$
- subtract $ytrue$
- square the result

### Q1

Fill in the **output** and **$f'(g(x))$** columns of the following table.
Remember when calculating the output column that the input to each layer is
the output of the previous layer.  Also remember that `g(x)` just refers to the
output of the previous layer.

following table representing the forward pass (that is, where you
calculate outputs of each sequential layer).  Let `x = 3`, `ytrue
= 5`, and `m = 7`.

| layer                 |  `f(n)`        | output | `f'(n)` | `f'(g(x))` |
| --------------------- | -------------- | ------ | ------- | ---------- |
| start with $x = 3$    |   n/a          |   1    |   n/a   |    n/a     |
| multiply by $m = 7$   |  $m \cdot n$   |        |   $m$   |            |
| subtract $ytrue = 5$  |  $n-5$         |        |   $1$   |            |
| square the result     |  $n^2$         |        |   $2n$  |            |

And lastly, what is the product of all the numbers in the final column?

### A Complete Example, In PyTorch

Now let's have PyTorch do the same thing for us.

Start with some tensors `x` and `y`.

```python
import torch
x = torch.tensor([3.0], requires_grad=True)
y = torch.tensor([5.0], requires_grad=True)
```

Then create a trainable tensor `m`.  "Trainable" means that we want PyTorch
to calculate gradients during backpropagation, which we can use to modify the
tensor.

```python
m = torch.tensor([7.0], requires_grad=True)
```

Now do a manual forward pass, feeding the output of each layer to the input of
the next, and print the results.  Then ask PyTorch to do a backward pass by
calling `loss.backward()`, which will set an `m.grad` variable representing the
gradient for `m`.

```python
print("-- forward pass --")
mx = x * m
err = mx - y
loss = err.square()

print("m initial:", m)
print("mx forward:", mx)
print("err forward:", err)
print("loss forward:", loss)

#
# Q2 code will go here in just a bit
#

print("-- backward pass --")
loss.backward()
print(f"-- final gradient: {m.grad} --")
```

Make sure that the forward pass numbers match the output column you calculated
in Q1, and the final gradient matches the product of the numbers in your final
column from Q1.

### Q2

Write code to trace the calculation of `m.grad` by adding backward hooks to
each of the following tensors:

- `m`
- `mx`
- `err`
- `loss`

Use this as an excuse to get familiar with PyTorch internals if you aren't
already familiar.  I suggest checking out the PyTorch codebase to your machine,
then browsing via your IDE or `git grep` to find where the `torch.Tensor` class
is defined, then looking around in there.

What order do the backward hooks print, and what are the gradients at each
step?  Explain what you see happening.

### Q3

If you rerun your forward pass (don't reinitialize `m`, but start with
`mx = x * m`), then rerun `loss.backward()`, what effect does it have on your
`m.grad`?

Fun fact: multiple forward passes and `backward()` calls without zeroing
gradients in between is exactly how we implement gradient aggregation in
`PyTorchTrial` (only we don't reuse the same data for both forward passes).

## Optimizers

In the previous exercise, after calculating the gradients, we applied them to
the model with the following formula:

    m -= gradient * lr

In deep learning frameworks, you don't normally update model weights directly
like that.  Normally, updating the model is the job of the optimizer.

In PyTorch, the update formula we applied so far is implemented by the SGD
optimizer:

```python
# create an SGD optimzer to apply gradients to our model weights (just m):
opt = torch.optim.SGD([m], lr=0.001)

# since we already called loss.backward() above, we can step our optimizer to
# apply those gradients, with the configured lr, to our model:
opt.step()
print(f"m after step: {m}")
```

### Q4:

What is the value of `m` after running `opt.step()`?  Can you confirm that it
matches the gradient update formula we used?

### More Sophisticated Optimizers

The gradient update formula we have used so far is pretty simple.  This simple
strategy is limited by some drawbacks:

- If you ever get stuck in a local minimum, the gradient may be zero even
  though there is a much better loss that could be obtained with different
  model parameters.  SGD will leave you in the local minimum forever.

- It may be unduly affected by bad data, causing you to keep a lower learning
  rate (and take longer to train) to compensate.

There are many more sophisticated gradient updates in existence.  We're going
to discuss a simple addition to the basic formula, which is to add "momentum"
to our gradient calculation.  Momentum means that the gradient we apply at each
step is a combination of the gradient we calculated in that step and the
gradient we calculated in the previous step.  The formula becomes:

```math
G_{n} = \mu G_{n-1} + g_{n}
```

Where:
- $g_{n}$ is the calculated gradient for step $n$
- $G_{n}$ is the momentum-adjusted gradient at step $n$
- $\mu$ is the momentum constant.

Momentum helps to solve the above-mentioned shortcomings because:

- When training approaches a local minimum, there's a possibility that the
  momentum of our gradient will cause us to "roll" through the minimum without
  stopping (and becoming stuck).

- When we process bad data, the resulting gradient will steer training in the
  wrong direction, but the effect it has is reduced if we've been processing
  other good data in recent batches.

There's a more detailed explanation with some illustrations [here](
https://optimization.cbe.cornell.edu/index.php?title=Momentum)

### Q4

If you have a model with N parameters, how many parameters are in the
calculated gradient?  And how many parameters are in the running average
gradient?  And what is the largest number of parameters you would have to fit
in memory at any given time to use a optimizer using the momentum formula?

### Q5

Go read the abstract, the introduction, and the update formula from the paper
describing [popular Adam optimizer](https://arxiv.org/abs/1412.6980).  Notice
what they call the "first moment" is similar in formula to momentum.  With a
model of N parameters, what is the largest number of parameters you would have
to fit into memory to use an Adam optimizer?

### Q6

Write a training loop that implements momentum manually.  Use direct Tensor
manipulation rather than high-level APIs; part of the goal is to learn some
PyTorch internals you may not have known before.  Your solution should not
include an actual torch optimizer object, but you should use the torch's `SGD`
optimizer as your ground truth.  Make sure your implementation is within
floating-point error of the output of PyTorch's momentum implementation.

Here's a template to get you started:

```python
import torch

LR = 0.01
MOMENTUM = 0.9

# some bogus model with a pair of 5x5 tensors
l1 = torch.nn.Linear(5, 5, bias=False)
l2 = torch.nn.Linear(5, 5, bias=False)
model = torch.nn.Sequential(l1, l2)

mse = torch.nn.MSELoss()

# the optimizer uses momentum
opt = torch.optim.SGD(model.parameters(), lr = LR, momentum=MOMENTUM)

def pytorch_train_batch(data):
    pred = model(data)
    # pretend our input is also our label
    loss = mse(pred, data)
    loss.backward()
    opt.step()
    opt.zero_grad()
    return loss.item()

# steal a copy of the randomly-initialized weights
my_l1 = torch.Tensor(l1.weight.detach().clone())
my_l2 = torch.Tensor(l2.weight.detach().clone())
my_l1.requires_grad = True
my_l.requires_grad = True

def my_train_batch(data):
    pred = my_l2.matmul(my_l1.matmul(data))
    loss = mse(pred, data)
    loss.backward()

    #####################
    # your code goes here
    #####################

# test your momentum implementation against pytorch's
for i in range(100):
    data = torch.rand(5)
    ptloss = pytorch_train_batch(data)
    myloss = my_train_batch(data)
    assert abs(ptloss - myloss) < 0.000001, (
        f"i = {i}, ptloss = {ptloss}, myloss = {myloss}"
    )
```

## TensorFlow

While we do have more PyTorch users than TensorFlow/Keras users, but we develop
features and field support questions from both frameworks.  We also maintain a
Keras-based feature in the upstream Horovod project.  So it is important to be
familiar with both.

### Q7

Train a model using momentum again, this time in TensorFlow.  As before, your
solution should use low-level Tensor manipulations rather than high-level APIs.
You might start reading about [tf.Variable](
https://www.tensorflow.org/guide/variable) and [tf.GradientTape](
https://www.tensorflow.org/guide/autodiff).

Note that tf.Variable and tf.GradientTape will come up in OSS support, whenever
users subclass `keras.models.Model` or `keras.metrics.Metric`.

Here's a template to get you started:

```python
import tensorflow as tf
from tensorflow.keras import layers, losses, models, optimizers

LR = 0.01
MOMENTUM = 0.9

# same bogus model as with the pytorch example
model = models.Sequential()
model.add(layers.Dense(5, activation=None, use_bias=False, input_shape=(5,)))
model.add(layers.Dense(5, activation=None, use_bias=False))
opt = optimizers.SGD(learning_rate=LR, momentum=MOMENTUM)
model.compile(
    optimizers.SGD(learning_rate=LR, momentum=MOMENTUM),
    losses.mean_squared_error,
)

# Your code here:
# - copy initial weights
# - define my_train_batch()

# test your momentum implementation against tensorflow's
for i in range(100):
    data = tf.random.uniform((1, 5))
    tfloss = model.train_on_batch(data, data)
    myloss = my_train_iter(data)
    assert abs(tfloss - myloss) < 0.000001, (
        f"i = {i}, tfloss = {tfloss}, myloss = {myloss}"
    )
```

## Optimizers and Large Models

As you have seen, optimizer state is often at least as large as model state,
even for fairly optimizers strategies like momentum.  This effect is compounded
with other advanced features like mixed-precision training, where mixed
precision, where even though model parameters require less memory each, the
per-parameter optimizer state memory requirement actually grows.

As a result, training large models often means dealing with very large
optimizer states.  A popular strategy is the [ZeRO Memory Optimization](
https://arxiv.org/abs/1910.02054).

### Q8

Go read the [ZeRO Memory Optimization](https://arxiv.org/abs/1910.02054) paper.
Read at least the intro and section 5.1 about Optimizer State Partitioning
($P_{os}$).  Then take the momentum training loop you wrote  and write a training
loop that simulates momentum-based training with Optimizer State Partitioning
across two workers.  Feel free to do it in PyTorch or TensorFlow.

No need for actual distributed mechanics; just do it inside a single process.
But do make sure your code demonstrates how the optimizer state can be sharded
so each worker only needs to store half of the momentum state.

## Summary

Upon completion of this exercise, you have learned:

- What optimizers do, and implemented a momentum optimizer yourself.
- Familiarized yourself with low-level PyTorch and TensorFlow code.
- Implemented one stage of ZeRO-DP memory optimization yourself.
