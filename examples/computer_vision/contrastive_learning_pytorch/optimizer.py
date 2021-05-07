"""
LARS: Layer-wise Adaptive Rate Scaling

Ported from TensorFlow to PyTorch
https://github.com/google-research/simclr/blob/master/lars_optimizer.py

License for the origin tf code for simclr is reproduced below.

=============================================================================
Copyright 2020 The SimCLR Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific simclr governing permissions and
limitations under the License.
"""

import torch
from torch.optim import optimizer


def lars(
    params,
    d_p_list,
    momentum_buffer_list,
    weight_decay,
    momentum,
    lr,
    eeta,
    classic_momentum,
    use_nesterov,
):
    for i, param in enumerate(params):
        d_p = d_p_list[i]
        if weight_decay != 0:
            d_p = d_p.add(param, alpha=weight_decay)

        if classic_momentum:
            buf = momentum_buffer_list[i]

            # Compute scaled_lr
            trust_ratio = 1.0
            w_norm = torch.norm(param)
            g_norm = torch.norm(d_p)

            w_norm = w_norm.item()
            g_norm = g_norm.item()

            if g_norm > 0 and w_norm > 0:
                trust_ratio = eeta * w_norm / g_norm

            scaled_lr = lr * trust_ratio

            buf.mul_(momentum).add_(d_p, alpha=scaled_lr)

            if use_nesterov:
                update = (momentum * buf) + (scaled_lr * d_p)
            else:
                update = buf

            param.add_(-update)
        else:
            raise NotImplementedError


class LARS(optimizer.Optimizer):
    """
    Layer-wise Adaptive Rate Scaling for large batch training.
    Introduced by "Large Batch Training of Convolutional Networks" by Y. You,
    I. Gitman, and B. Ginsburg. (https://arxiv.org/abs/1708.03888)
    """

    def __init__(
        self,
        params,
        lr=1.0,
        momentum=0.9,
        use_nesterov=False,
        weight_decay=0.0,
        classic_momentum=True,
        eeta=0.001,
    ):
        """
        Args:
            lr: A `float` for learning rate.
            momentum: A `float` for momentum.
            use_nesterov: A 'Boolean' for whether to use nesterov momentum.
            weight_decay: A `float` for weight decay.
            classic_momentum: A `boolean` for whether to use classic (or popular)
                momentum. The learning rate is applied during momeuntum update in
                classic momentum, but after momentum for popular momentum.
            eeta: A `float` for scaling of learning rate when computing trust ratio.
                name: The name for the scope.
        """

        defaults = dict(
            lr=lr,
            momentum=momentum,
            use_nesterov=use_nesterov,
            weight_decay=weight_decay,
            classic_momentum=classic_momentum,
            eeta=eeta,
        )

        super(LARS, self).__init__(params, defaults)

    def __setstate__(self, state):
        super(LARS, self).__setstate__(state)

    @torch.no_grad()
    def step(self, closure=None):
        loss = None
        if closure is not None:
            with torch.enable_grad():
                loss = closure()

        for group in self.param_groups:
            params_with_grad = []
            d_p_list = []
            momentum_buffer_list = []
            weight_decay = group["weight_decay"]
            momentum = group["momentum"]
            eeta = group["eeta"]
            lr = group["lr"]
            classic_momentum = group["classic_momentum"]
            use_nesterov = group["use_nesterov"]

            for p in group["params"]:
                if p.grad is not None:
                    params_with_grad.append(p)
                    d_p_list.append(p.grad)

                    state = self.state[p]
                    if "momentum_buffer" not in state:
                        momentum_buffer_list.append(torch.zeros_like(p))
                    else:
                        momentum_buffer_list.append(state["momentum_buffer"])

            lars(
                params_with_grad,
                d_p_list,
                momentum_buffer_list,
                weight_decay,
                momentum,
                lr,
                eeta,
                classic_momentum,
                use_nesterov,
            )

            # update momentum_buffers in state
            for p, momentum_buffer in zip(params_with_grad, momentum_buffer_list):
                state = self.state[p]
                state["momentum_buffer"] = momentum_buffer

        return loss
