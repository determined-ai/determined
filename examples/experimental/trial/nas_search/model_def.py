"""
This example is to shows a possible way to create a random NAS search using Determined.
The flags and configurations can be found under const.yaml and random.yaml.

Using Determined, we are able to skip step 1 because we can search using Determined capabilities.

This implementation is based on:
https://github.com/liamcli/randomNAS_release/tree/6513a0a6a781ed1f0009ccd9bae622ae7f0a961d

Paper for reference: https://arxiv.org/pdf/1902.07638.pdf

"""
import logging
import math
import os
import pickle as pkl
from typing import Dict, Sequence, Union

import numpy as np
import randomNAS_files.genotypes as genotypes
import randomNAS_files.data_util as data_util
import torch
from randomNAS_files.model import RNNModel
from torch import nn
from torch.optim.lr_scheduler import _LRScheduler

from determined.pytorch import ClipGradsL2Norm, DataLoader, PyTorchCallback, PyTorchTrial, PyTorchTrialContext, LRScheduler


import data

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]
PTB_NUMBER_TOKENS = 10000


class MyLR(_LRScheduler):
    def __init__(self, optimizer, hparams, last_epoch=-1):
        """
        Custom LR scheudler for the LR to be adjusted based on the batch size
        """
        self.hparams = hparams
        self.seq_len = hparams["bptt"]
        self.start_lr = hparams["learning_rate"]
        super(MyLR, self).__init__(optimizer, last_epoch)

    def get_lr(self):
        ret = list(self.base_lrs)
        self.base_lrs = [
            self.start_lr * self.seq_len / self.hparams["bptt"] for base_lr in self.base_lrs
        ]
        return ret

    def set_seq_len(self, seq_len):
        self.seq_len = seq_len


class NASModel(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context

        # Create a unique download directory for each rank so they don't overwrite each other.
        self.download_directory = f"/tmp/data-rank{self.context.distributed.get_rank()}"
        self.data_downloaded = False

        # Initialize the model
        arch_to_use = self.context.get_hparam("arch_to_use")

        if hasattr(genotypes, arch_to_use):
            self.arch = getattr(genotypes, arch_to_use)
            logging.info("using genotype.{0}".format(self.arch))
        else:
            self.arch = self.sample_arch()
            logging.info("using random arch.{0}".format(self.arch))

        model = RNNModel(
            PTB_NUMBER_TOKENS,
            self.context.get_hparam("emsize"),
            self.context.get_hparam("nhid"),
            self.context.get_hparam("nhidlast"),
            self.context.get_hparam("dropout"),
            self.context.get_hparam("dropouth"),
            self.context.get_hparam("dropoutx"),
            self.context.get_hparam("dropouti"),
            self.context.get_hparam("dropoute"),
            genotype=self.arch,
        )

        # Made for stacking multiple cells, by default the depth is set to 1
        # which will not run this for loop
        for _ in range(
                self.context.get_hparam("depth") - 1
        ):  # minus 1 because 1 gets auto added by the main model
            new_cell = model.cell_cls(
                self.context.get_hparam("emsize"),
                self.context.get_hparam("nhid"),
                self.context.get_hparam("dropouth"),
                self.context.get_hparam("dropoutx"),
                self.arch,
                self.context.get_hparam("init_op"),
            )
            model.rnns.append(new_cell)

        model.batch_size = self.context.get_per_slot_batch_size()
        self.model = self.context.wrap_model(model)
        self.optimizer = self.context.wrap_optimizer(torch.optim.SGD(
            self.model.parameters(),
            lr=self.context.get_hparam("learning_rate"),
            weight_decay=self.context.get_hparam("wdecay"),
        ))

        myLR = MyLR(self.optimizer, self.context.get_hparams())
        step_mode = LRScheduler.StepMode.MANUAL_STEP
        if self.context.get_hparam("step_every_batch"):
            step_mode = LRScheduler.StepMode.STEP_EVERY_BATCH
        elif self.context.get_hparam("step_every_epoch"):
            step_mode = LRScheduler.StepMode.STEP_EVERY_EPOCH
        self.myLR = self.context.wrap_lrscheduler(myLR, step_mode=step_mode)

    def sample_arch(self):
        """
        Required: Method to build the Optimizer
        Returns: PyTorch Optimizer
        """
        n_nodes = genotypes.STEPS
        n_ops = len(genotypes.PRIMITIVES)
        arch = []
        for i in range(n_nodes):
            op = np.random.choice(range(1, n_ops))
            node_in = np.random.choice(range(i + 1))
            arch.append((genotypes.PRIMITIVES[op], node_in))
        concat = range(1, 9)
        genotype = genotypes.Genotype(recurrent=arch, concat=concat)
        return genotype

    def update_and_step_lr(self, seq_len):
        """
        Updates and steps the learning rate
        """
        self.myLR.set_seq_len(seq_len)
        self.myLR.step()

    def train_batch(self, batch: TorchData, epoch_idx: int, batch_idx: int):
        """
        Trains the provided batch.
        Returns: Dictionary of the calculated Metrics
        """

        features, labels = batch
        self.update_and_step_lr(features.shape[0])

        # set hidden if it's the first run
        if batch_idx == 0:
            self.hidden = self.model.init_hidden(self.context.get_per_slot_batch_size())

        # detach to prevent backpropagating to far
        for i in range(len(self.hidden)):
            self.hidden[i] = self.hidden[i].detach()

        log_prob, self.hidden, rnn_hs, dropped_rnn_hs = self.model(features, self.hidden, return_h=True)

        loss = nn.functional.nll_loss(
            log_prob.view(-1, log_prob.size(2)), labels.contiguous().view(-1)
        )
        if self.context.get_hparam("alpha") > 0:
            loss = loss + sum(
                self.context.get_hparam("alpha") * dropped_rnn_h.pow(2).mean()
                for dropped_rnn_h in dropped_rnn_hs[-1:]
            )

        loss = (
            loss
            + sum(
                self.context.get_hparam("beta") * (rnn_h[1:] - rnn_h[:-1]).pow(2).mean()
                for rnn_h in rnn_hs[-1:]
            )
        ) * 1.0

        try:
            perplexity = math.exp(loss / len(features))
        except Exception as e:
            logging.error("Calculating perplexity failed with error: %s", e)
            perplexity = 100000

        if math.isnan(perplexity):
            perplexity = 100000

        self.context.backward(loss)
        self.context.step_optimizer(
            self.optimizer,
            clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(
                params, self.context.get_hparam("clip_gradients_l2_norm")
            )
        )

        return {"loss": loss, "perplexity": perplexity}

    def evaluate_full_dataset(self, data_loader: torch.utils.data.DataLoader):
        """
        Determines if multiple architectures should be evaluated and sends to approprate path
        Returns: the results of the evaluated dataset or the best result from multiple evaluations
        """
        eval_same_arch = self.context.get_hparam("eval_same_arch")

        if eval_same_arch:  # evaluate the same architecture
            res = self.evaluate_dataset(data_loader, self.arch)
        else:
            res = self.evaluate_multiple_archs(data_loader)

        return res

    def evaluate_multiple_archs(self, data_loader):
        """
        Helper that randomly selects architectures and evaluates their performance
        This function is only called if eval_same_arch is False and should not be used for
        the primary NAS search
        """
        num_archs_to_eval = self.context.get_hparam("num_archs_to_eval")

        sample_vals = []
        for _ in range(num_archs_to_eval):
            arch = self.sample_arch()

            res = self.evaluate_dataset(data_loader, arch)
            perplexity = res["perplexity"]
            loss = res["loss"]

            sample_vals.append((arch, perplexity, loss))

        sample_vals = sorted(sample_vals, key=lambda x: x[1])

        logging.info("best arch found: ", sample_vals[0])
        self.save_archs(sample_vals)

        return {"loss": sample_vals[0][2], "perplexity": sample_vals[0][1]}

    def evaluate_dataset(self, data_loader, arch, split=None):
        """
        Evaluates the full dataset against the given arch
        """
        hidden = self.model.init_hidden(self.context.get_hparam("eval_batch_size"))

        model = self.set_model_arch(arch, self.model)

        total_loss = 0
        num_samples_seen = 0
        for i, batch in enumerate(data_loader):
            features, targets = batch
            features, targets = features.cuda(), targets.cuda()

            log_prob, hidden = model(features, hidden)
            loss = nn.functional.nll_loss(log_prob.view(-1, log_prob.size(2)), targets).data
            total_loss += loss * len(features)

            for i in range(len(hidden)):
                hidden[i] = hidden[i].detach()
            num_samples_seen += features.shape[0]

        try:
            perplexity = math.exp(total_loss.item() / num_samples_seen)
        except Exception as e:
            logging.error("Calculating perplexity failed with error: %s", e)
            perplexity = 100000

        if math.isnan(perplexity):
            perplexity = 100000

        if math.isnan(loss):
            loss = 100000

        return {"loss": total_loss, "perplexity": perplexity}

    def save_archs(self, data):
        out_file = self.context.get_data_config().get("out_file") + self.context.get_hparam("seed")

        with open(os.path.join(out_file), "wb+") as f:
            pkl.dump(data, f)

    def set_model_arch(self, arch, model):
        for rnn in model.rnns:
            rnn.genotype = arch
        return model

    def build_training_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            data.download_data(self.download_directory)
            self.data_downloaded = True

        corpus = data_util.Corpus(self.download_directory)

        train_dataset = data.PTBData(
            corpus.train,
            self.context.get_hparam("seq_len"),
            self.context.get_per_slot_batch_size(),
            self.context.get_hparam("bptt"),
            self.context.get_hparam("max_seq_length_delta"),
        )
        return DataLoader(
            train_dataset,
            batch_sampler=data.BatchSamp(
                train_dataset,
                self.context.get_hparam("bptt"),
                self.context.get_hparam("max_seq_length_delta"),
            ),
            collate_fn=data.PadSequence(),
        )

    def build_validation_data_loader(self) -> DataLoader:
        if not self.data_downloaded:
            data.download_data(self.download_directory)
            self.data_downloaded = True

        corpus = data_util.Corpus(self.download_directory)

        test_dataset = data.PTBData(
            corpus.valid,
            self.context.get_hparam("seq_len"),
            self.context.get_hparam("eval_batch_size"),
            self.context.get_hparam("bptt"),
            self.context.get_hparam("max_seq_length_delta"),
        )

        return DataLoader(
            test_dataset,
            batch_sampler=data.BatchSamp(
                test_dataset,
                self.context.get_hparam("bptt"),
                self.context.get_hparam("max_seq_length_delta"),
                valid=True,
            ),
            collate_fn=data.PadSequence(),
        )

