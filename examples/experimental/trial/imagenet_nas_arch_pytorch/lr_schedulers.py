import torch
from torch.optim.lr_scheduler import _LRScheduler
from torch.optim.sgd import SGD


# These LR schedulers are modified to use a warmup period with linearly increasing LR.
def WarmupWrapper(scheduler_type):
    class Wrapped(scheduler_type):
        def __init__(self, warmup_epochs, *args):
            self.warmup_epochs = warmup_epochs
            super(Wrapped, self).__init__(*args)

        def get_lr(self):
            if self.last_epoch < self.warmup_epochs:
                return [
                    (self.last_epoch + 1) / self.warmup_epochs * b_lr
                    for b_lr in self.base_lrs
                ]
            return super(Wrapped, self).get_lr()

    return Wrapped


class LinearLRScheduler(_LRScheduler):
    def __init__(self, optimizer, max_epochs, warmup_epochs, last_epoch=-1):
        self.optimizer = optimizer
        self.warmup_epochs = warmup_epochs
        self.max_epochs = max_epochs
        self.last_epoch = last_epoch
        super(LinearLRScheduler, self).__init__(optimizer, last_epoch)

    def get_lr(self):
        if self.max_epochs - self.last_epoch > self.warmup_epochs:
            lr_mult = (self.max_epochs - self.warmup_epochs - self.last_epoch) / (
                self.max_epochs - self.warmup_epochs
            )
        else:
            # We use very small lr for last few epochs to help convergence.
            lr_mult = (self.max_epochs - self.last_epoch) / (
                (self.last_epoch - self.warmup_epochs) * 5
            )
        return [base_lr * lr_mult for base_lr in self.base_lrs]


class EfficientNetScheduler(_LRScheduler):
    def __init__(self, optimizer, gamma, decay_every, last_epoch=-1):
        self.optimizer = optimizer
        self.last_epoch = last_epoch
        self.gamma = gamma
        self.decay_every = decay_every
        super(EfficientNetScheduler, self).__init__(optimizer, last_epoch)

    def get_lr(self):
        lr_mult = self.gamma ** (int((self.last_epoch + 1) / self.decay_every))
        return [base_lr * lr_mult for base_lr in self.base_lrs]
