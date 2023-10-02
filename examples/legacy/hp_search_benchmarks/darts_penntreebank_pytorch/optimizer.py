import logging

import torch
from torch.optim.optimizer import Optimizer


class HybridSGD(Optimizer):
    def __init__(
        self,
        params,
        # shared params
        lr=0,
        weight_decay=0,
        # SGD params
        momentum=0,
        dampening=0,
        nesterov=False,
        # ASGD params
        lambd=1e-4,
        alpha=0.75,
        t0=1e6,
    ):
        """
        This optimizer wraps around SGD or ASGD and allows us to switch between
        these optimizers.
        """
        defaults = dict(
            lr=lr,
            weight_decay=weight_decay,
            momentum=momentum,
            dampening=dampening,
            nesterov=nesterov,
            lambd=lambd,
            alpha=alpha,
            t0=t0,
        )
        self.defaults = defaults
        params = list(params)
        super(HybridSGD, self).__init__(params, defaults)
        self.SGD = torch.optim.SGD(params, lr, momentum, dampening, weight_decay, nesterov)
        self.ASGD = torch.optim.ASGD(params, lr, lambd, alpha, t0, weight_decay)

        # Always initialize optimizer to use SGD.
        # If we are resuming optimizer state, will change optimizer
        # based on the resume state.
        self.optim_name = "SGD"
        self.optim = self.SGD

    def set_optim(self, optim_name):
        assert optim_name in ("SGD", "ASGD")
        self.optim_name = optim_name
        if optim_name == "SGD":
            self.optim = self.SGD
        elif optim_name == "ASGD":
            self.optim = self.ASGD

    def state_dict(self):
        return self.optim.state_dict()

    def load_state_dict(self, state_dict):
        # We know the saved optimizer state is for ASGD if t0
        # is in a field of the param dict.
        if "t0" in state_dict["param_groups"][0]:
            logging.info("Resuming ASGD optimizer")
            self.optim_name = "ASGD"
            self.optim = self.ASGD
        self.optim.load_state_dict(state_dict)

    def step(self, closure=None):
        self.optim.step(closure=closure)


if __name__ == "__main__":
    w = torch.randn(1, requires_grad=True)
    loss = torch.nn.MSELoss()
    optimizer = HybridSGD([w], 0.1, 0.0004)
    for _ in range(10):
        x = torch.randn(100)
        y = 3 * x
        output = loss(w * x, y)
        output.backward()
        optimizer.step()
        optimizer.zero_grad()
    # Should return SGD state_dict
    print(optimizer.state_dict())
    print(w)
    optimizer.set_optim("ASGD")
    for _ in range(20):
        x = torch.randn(100)
        y = 3 * x
        output = loss(w * x, y)
        output.backward()
        optimizer.step()
        optimizer.zero_grad()
    # Should return ASGD state_dict with additional
    # keys in param_groups.
    print(optimizer.state_dict())
    print(w)
