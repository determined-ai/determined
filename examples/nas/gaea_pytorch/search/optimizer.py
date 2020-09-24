import torch
from torch.optim.optimizer import Optimizer, required


class EG(Optimizer):
    def __init__(self, params, lr=required, normalize_fn=lambda x: x):
        if lr is not required and lr < 0.0:
            raise ValueError("Invalid learning rate: {}".format(lr))
        self.normalize_fn = normalize_fn
        defaults = dict(lr=lr)
        super(EG, self).__init__(params, defaults)

    @torch.no_grad()
    def step(self, closure=None):
        loss = None
        if closure is not None:
            with torch.enable_grad():
                loss = closure()

        for group in self.param_groups:
            for p in group["params"]:
                if p.grad is None:
                    continue
                d_p = p.grad
                p.mul_(torch.exp(-group["lr"] * d_p))
                p.data = self.normalize_fn(p.data)

        return loss
