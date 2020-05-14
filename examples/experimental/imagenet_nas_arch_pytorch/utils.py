from collections import namedtuple
from typing import Any

import numpy as np
import torch
from torch import nn

import autoaugment.augmentation_transforms as augmentation_transforms
import autoaugment.policies as found_policies

# From: https://github.com/quark0/DARTS
Genotype = namedtuple("Genotype", "normal normal_concat reduce reduce_concat")


def accuracy(output, target, topk=(1,)):
    maxk = max(topk)
    batch_size = target.size(0)

    _, pred = output.topk(maxk, 1, True, True)
    pred = pred.t()
    correct = pred.eq(target.view(1, -1).expand_as(pred))

    res = []
    for k in topk:
        correct_k = correct[:k].view(-1).float().sum(0)
        res.append(correct_k.mul_(100.0 / batch_size))
    return res


def drop_path(x, drop_prob):
    if drop_prob > 0.0:
        keep_prob = 1.0 - drop_prob
        mask = Variable(
            torch.cuda.FloatTensor(x.size(0), 1, 1, 1).bernoulli_(keep_prob)
        )
        x.div_(keep_prob)
        x.mul_(mask)
    return x


class Cutout(object):
    def __init__(self, length):
        self.length = length

    def __call__(self, img):
        h, w = img.size(1), img.size(2)
        mask = np.ones((h, w), np.float32)
        y = np.random.randint(h)
        x = np.random.randint(w)

        y1 = np.clip(y - self.length // 2, 0, h)
        y2 = np.clip(y + self.length // 2, 0, h)
        x1 = np.clip(x - self.length // 2, 0, w)
        x2 = np.clip(x + self.length // 2, 0, w)

        mask[y1:y2, x1:x2] = 0.0
        mask = torch.from_numpy(mask)
        mask = mask.expand_as(img)
        img *= mask
        return img


# From: https://github.com/yuhuixu1993/PC-DARTS
class CrossEntropyLabelSmooth(nn.Module):
    """
    Assign small probability to non-target classes to hopefully learn faster and more generalizable features.

    See this paper for more info:
    https://arxiv.org/pdf/1906.02629.pdf
    """

    def __init__(self, num_classes, epsilon):
        super(CrossEntropyLabelSmooth, self).__init__()
        self.num_classes = num_classes
        self.epsilon = epsilon
        self.logsoftmax = nn.LogSoftmax(dim=1)

    def forward(self, inputs, targets):
        log_probs = self.logsoftmax(inputs)
        targets = torch.zeros_like(log_probs).scatter_(1, targets.unsqueeze(1), 1)
        targets = (1 - self.epsilon) * targets + self.epsilon / self.num_classes
        loss = (-targets * log_probs).mean(0).sum()
        return loss


# Memory efficient version for training from: https://github.com/lukemelas/EfficientNet-PyTorch/blob/master/efficientnet_pytorch/utils.py
class SwishImplementation(torch.autograd.Function):
    @staticmethod
    def forward(ctx, i):
        result = i * torch.sigmoid(i)
        ctx.save_for_backward(i)
        return result

    @staticmethod
    def backward(ctx, grad_output):
        i = ctx.saved_variables[0]
        sigmoid_i = torch.sigmoid(i)
        return grad_output * (sigmoid_i * (1 + i * (1 - sigmoid_i)))


class Swish(nn.Module):
    """Swish activation function.
    See: https://arxiv.org/abs/1710.05941
    """

    def forward(self, x):
        return SwishImplementation.apply(x)


class HSwish(nn.Module):
    """Hard Swish activation function.
    See: https://arxiv.org/abs/1905.02244
    """

    def forward(self, x):
        return x * nn.functional.relu6(x + 3).div_(6)


def AutoAugment(img):
    """
    Augmentation policy learned by RL.  From:
        https://arxiv.org/abs/1805.09501
    """
    good_policies = found_policies.good_policies()
    policy = good_policies[np.random.choice(len(good_policies))]
    final_img = augmentation_transforms.apply_policy(policy, img)
    return final_img


class SqueezeAndExcitation(nn.Module):
    """Squeeze-and-Excitation module.
    See: https://arxiv.org/abs/1709.01507
    """

    def __init__(self, n_feature, n_hidden, spatial_dims=[2, 3], active_fn=None):
        super(SqueezeAndExcitation, self).__init__()
        self.n_feature = n_feature
        self.n_hidden = n_hidden
        self.spatial_dims = spatial_dims
        self.se_reduce = nn.Conv2d(n_feature, n_hidden, 1, bias=True)
        self.se_expand = nn.Conv2d(n_hidden, n_feature, 1, bias=True)
        self.active_fn = active_fn()

    def forward(self, x):
        se_tensor = x.mean(self.spatial_dims, keepdim=True)
        se_tensor = self.se_expand(self.active_fn(self.se_reduce(se_tensor)))
        return torch.sigmoid(se_tensor) * x

    def __repr__(self):
        return "{}({}, {}, spatial_dims={}, active_fn={})".format(
            self._get_name(),
            self.n_feature,
            self.n_hidden,
            self.spatial_dims,
            self.active_fn,
        )
