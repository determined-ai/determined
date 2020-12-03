from functools import partial

import torch
import torch.nn as nn
from torch.autograd import Variable

from operations import *
from utils import SqueezeAndExcitation, drop_path


class Cell(nn.Module):
    def __init__(
        self,
        genotype,
        C_prev_prev,
        C_prev,
        C,
        reduction,
        reduction_prev,
        activation_function=nn.ReLU,
        drop_prob=0,
    ):
        super(Cell, self).__init__()
        print(C_prev_prev, C_prev, C)

        self.drop_prob = drop_prob
        self.drop_module = nn.Dropout2d(drop_prob)

        if reduction_prev:
            self.preprocess0 = FactorizedReduce(C_prev_prev, C)
        else:
            self.preprocess0 = ActivationConvBN(
                activation_function, C_prev_prev, C, 1, 1, 0
            )
        self.preprocess1 = ActivationConvBN(activation_function, C_prev, C, 1, 1, 0)

        if reduction:
            op_names, indices = zip(*genotype.reduce)
            concat = genotype.reduce_concat
        else:
            op_names, indices = zip(*genotype.normal)
            concat = genotype.normal_concat
        self._compile(C, op_names, indices, concat, reduction, activation_function)

    def _compile(self, C, op_names, indices, concat, reduction, activation_function):
        assert len(op_names) == len(indices)
        self._steps = len(op_names) // 2
        self._concat = concat
        self.multiplier = len(concat)

        self._ops = nn.ModuleList()
        for name, index in zip(op_names, indices):
            stride = 2 if reduction and index < 2 else 1
            op = OPS[name](C, stride, True, activation_function)
            if "conv" in name and self.drop_prob > 0:
                op = nn.Sequential(self.drop_module, op)
            self._ops += [op]
        self._indices = indices

    def forward(self, s0, s1, drop_path_prob):
        s0 = self.preprocess0(s0)
        s1 = self.preprocess1(s1)

        states = [s0, s1]
        for i in range(self._steps):
            h1 = states[self._indices[2 * i]]
            h2 = states[self._indices[2 * i + 1]]
            op1 = self._ops[2 * i]
            op2 = self._ops[2 * i + 1]
            h1 = op1(h1)
            h2 = op2(h2)
            if self.training and drop_path_prob > 0.0:
                if not isinstance(op1, Identity):
                    h1 = drop_path(h1, drop_path_prob)
                if not isinstance(op2, Identity):
                    h2 = drop_path(h2, drop_path_prob)
            s = h1 + h2
            states += [s]
        return torch.cat([states[i] for i in self._concat], dim=1)


class AuxiliaryHeadImageNet(nn.Module):
    def __init__(
        self, C, num_classes, activation_function=partial(nn.ReLU, inplace=True)
    ):
        """assuming input size 14x14"""
        super(AuxiliaryHeadImageNet, self).__init__()
        self.features = nn.Sequential(
            activation_function(),
            nn.AvgPool2d(5, stride=2, padding=0, count_include_pad=False),
            nn.Conv2d(C, 128, 1, bias=False),
            nn.BatchNorm2d(128, momentum=0.999, eps=0.001),
            activation_function(),
            nn.Conv2d(128, 768, 2, bias=False),
            # NOTE: This batchnorm was omitted in my earlier implementation due to a typo.
            # Commenting it out for consistency with the experiments in the paper.
            nn.BatchNorm2d(768, momentum=0.999, eps=0.001),
            activation_function(),
        )
        self.classifier = nn.Linear(768, num_classes)

    def forward(self, x):
        x = self.features(x)
        x = self.classifier(x.contiguous().view(x.size(0), -1))
        return x


class NetworkImageNet(nn.Module):
    def __init__(
        self,
        genotype,
        activation_function,
        C,
        num_classes,
        layers,
        auxiliary,
        do_SE,
        drop_path_prob=0,
        drop_prob=0,
    ):
        super(NetworkImageNet, self).__init__()
        self._layers = layers
        self._auxiliary = auxiliary
        self._do_SE = do_SE
        self._drop_path_prob = drop_path_prob

        self.stem0 = nn.Sequential(
            nn.Conv2d(3, C // 2, kernel_size=3, stride=2, padding=1, bias=False),
            nn.BatchNorm2d(C // 2, momentum=0.999, eps=0.001),
            activation_function(),
            nn.Conv2d(C // 2, C, 3, stride=2, padding=1, bias=False),
            nn.BatchNorm2d(C, momentum=0.999, eps=0.001),
        )

        self.stem1 = nn.Sequential(
            activation_function(),
            nn.Conv2d(C, C, 3, stride=2, padding=1, bias=False),
            nn.BatchNorm2d(C, momentum=0.999, eps=0.001),
        )

        C_prev_prev, C_prev, C_curr = C, C, C

        self.cells = nn.ModuleList()
        self.cells_SE = nn.ModuleList()
        reduction_prev = True
        for i in range(layers):
            if i in [layers // 3, 2 * layers // 3]:
                C_curr *= 2
                reduction = True
            else:
                reduction = False
            cell = Cell(
                genotype,
                C_prev_prev,
                C_prev,
                C_curr,
                reduction,
                reduction_prev,
                activation_function,
                drop_prob,
            )
            reduction_prev = reduction
            self.cells += [cell]
            C_prev_prev, C_prev = C_prev, cell.multiplier * C_curr

            if self._do_SE and i <= layers * 2 / 3:
                if C_curr == C:
                    reduction_factor_SE = 4
                else:
                    reduction_factor_SE = 8
                self.cells_SE += [
                    SqueezeAndExcitation(
                        C_curr * 4,
                        C_curr // reduction_factor_SE,
                        active_fn=activation_function,
                    )
                ]
            if i == 2 * layers // 3:
                C_to_auxiliary = C_prev

        if auxiliary:
            self.auxiliary_head = AuxiliaryHeadImageNet(
                C_to_auxiliary, num_classes, activation_function=activation_function
            )
        self.global_pooling = nn.AvgPool2d(7)
        self.classifier = nn.Linear(C_prev, num_classes)

    def get_save_states(self):
        return {
            "state_dict": self.state_dict(),
        }

    def load_states(self, save_states):
        self.load_state_dict(save_states["state_dict"])

    def forward(self, input):
        logits_aux = None
        s0 = self.stem0(input)
        s1 = self.stem1(s0)
        for i, cell in enumerate(self.cells):
            s0, s1 = s1, cell(s0, s1, self._drop_path_prob)
            if self._do_SE and i <= len(self.cells) * 2 / 3:
                s1 = self.cells_SE[i](s1)
            if i == 2 * self._layers // 3:
                if self._auxiliary and self.training:
                    logits_aux = self.auxiliary_head(s1)
        out = self.global_pooling(s1)
        logits = self.classifier(out.contiguous().view(out.size(0), -1))
        return logits, logits_aux
