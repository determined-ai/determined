"""
This model is from the CNN NAS search space considered in:
    https://openreview.net/forum?id=S1eYHoC5FX

We will use the adaptive searcher in Determined to find a
good architecture in this search space for CIFAR-10.  
"""

import logging
import math
from collections import namedtuple
from typing import Any, Dict

import data
import randomNAS_files.data_util as data_util
import torch
from optimizer import HybridSGD
from randomNAS_files.model import RNNModel
from torch import nn
from torch.optim.lr_scheduler import _LRScheduler

from determined.pytorch import DataLoader, LRScheduler, PyTorchTrial, PyTorchTrialContext

Genotype = namedtuple("Genotype", "recurrent concat")


class AttrDict(dict):
    def __init__(self, *args, **kwargs):
        super(AttrDict, self).__init__(*args, **kwargs)
        self.__dict__ = self


class MyLR(_LRScheduler):
    def __init__(self, optimizer, hparams, last_epoch=-1):
        """
        Custom LR scheudler for the LR to be adjusted based on the batch size
        """
        self.hparams = hparams
        self.seq_len = hparams.bptt
        self.start_lr = hparams.learning_rate
        super(MyLR, self).__init__(optimizer, last_epoch)

    def get_lr(self):
        ret = list(self.base_lrs)
        self.base_lrs = [
            self.start_lr * self.seq_len / self.hparams.bptt for base_lr in self.base_lrs
        ]
        return ret

    def set_seq_len(self, seq_len):
        self.seq_len = seq_len


class DARTSRNNTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.data_config = context.get_data_config()
        self.hparams = AttrDict(context.get_hparams())

        # Create a unique download directory for each rank so they don't overwrite each
        # other when doing distributed training.
        self.download_directory = self.data_config["data_download_dir"]
        data.download_data(self.download_directory)
        corpus = data_util.Corpus(self.download_directory)
        self.corpus = corpus
        self.ntokens = len(corpus.dictionary)
        self.hidden = None

        # This is used to store eval history and will switch to ASGD
        # once validation perplexity stops improving.
        self._last_loss = None
        self._eval_history = []
        self._last_epoch = -1

        # Define the model
        genotype = self.get_genotype_from_hps()
        self.model = self.context.wrap_model(
            RNNModel(
                self.ntokens,
                self.hparams.emsize,
                self.hparams.nhid,
                self.hparams.nhidlast,
                self.hparams.dropout,
                self.hparams.dropouth,
                self.hparams.dropoutx,
                self.hparams.dropouti,
                self.hparams.dropoute,
                genotype=genotype,
            )
        )
        total_params = sum(x.data.nelement() for x in self.model.parameters())
        logging.info("Model total parameters: {}".format(total_params))

        # Define the optimizer
        self._optimizer = self.context.wrap_optimizer(
            HybridSGD(
                self.model.parameters(),
                self.hparams.learning_rate,
                self.hparams.weight_decay,
                lambd=0,
                t0=0,
            )
        )

        # Define the LR scheduler
        self.myLR = MyLR(self._optimizer, self.hparams)
        step_mode = LRScheduler.StepMode.MANUAL_STEP
        self.wrapped_LR = self.context.wrap_lr_scheduler(self.myLR, step_mode=step_mode)

    def build_training_data_loader(self) -> DataLoader:
        train_dataset = data.PTBData(
            self.corpus.train,
            self.context.get_per_slot_batch_size(),
            self.hparams.bptt,
            self.hparams.max_seq_length_delta,
        )
        return DataLoader(
            train_dataset,
            batch_sampler=data.BatchSamp(
                train_dataset,
                self.hparams.bptt,
                self.hparams.max_seq_length_delta,
            ),
            collate_fn=data.PadSequence(),
        )

    def build_validation_data_loader(self) -> DataLoader:
        test_dataset = data.PTBData(
            self.corpus.valid,
            self.hparams.eval_batch_size,
            self.hparams.bptt,
            self.hparams.max_seq_length_delta,
        )
        return DataLoader(
            test_dataset,
            batch_sampler=data.BatchSamp(
                test_dataset,
                self.hparams.bptt,
                self.hparams.max_seq_length_delta,
                valid=True,
            ),
            collate_fn=data.PadSequence(),
        )

    def get_genotype_from_hps(self):
        # This function creates an architecture definition
        # from the hyperparameter settings.
        cell_config = []
        for node in range(8):
            edge_ind = self.hparams["node{}_edge".format(node + 1)]
            edge_op = self.hparams["node{}_op".format(node + 1)]
            cell_config.append((edge_op, edge_ind))
        return Genotype(recurrent=cell_config, concat=range(1, 9))

    def update_and_step_lr(self, seq_len):
        """
        Updates and steps the learning rate
        """
        self.myLR.set_seq_len(seq_len)
        self.myLR.step()

    def switch_optimizer(self):
        if len(self._eval_history) > self.hparams.nonmono + 1:
            if self._last_loss > min(self._eval_history[: -(self.hparams.nonmono + 1)]):
                logging.info("Switching to ASGD.")
                self._optimizer.set_optim("ASGD")

    def train_batch(self, batch: Any, epoch_idx: int, batch_idx: int) -> Dict[str, torch.Tensor]:
        """
        Trains the provided batch.
        Returns: Dictionary of the calculated Metrics
        """
        if epoch_idx != self._last_epoch:
            logging.info("Starting epoch {}".format(epoch_idx))
            if (
                epoch_idx > self.hparams["optimizer_switch_epoch"]
                and self._optimizer.optim_name == "SGD"
            ):
                self.switch_optimizer()

        features, labels = batch
        self.update_and_step_lr(features.shape[0])

        # set hidden if it's the first run
        if batch_idx == 0 or self.hidden is None:
            self.hidden = self.model.init_hidden(self.context.get_per_slot_batch_size())

        # detach to prevent backpropagating to far
        for i in range(len(self.hidden)):
            self.hidden[i] = self.hidden[i].detach()

        log_prob, self.hidden, rnn_hs, dropped_rnn_hs = self.model(
            features, self.hidden, return_h=True
        )

        raw_loss = nn.functional.nll_loss(
            log_prob.contiguous().view(-1, log_prob.size(2)),
            labels.contiguous().contiguous().view(-1),
        )

        loss = raw_loss
        if self.hparams.alpha > 0:
            loss = loss + sum(
                self.hparams.alpha * dropped_rnn_h.pow(2).mean()
                for dropped_rnn_h in dropped_rnn_hs[-1:]
            )

        loss = (
            loss
            + sum(
                self.hparams.beta * (rnn_h[1:] - rnn_h[:-1]).pow(2).mean() for rnn_h in rnn_hs[-1:]
            )
        ) * 1.0

        try:
            perplexity = math.exp(raw_loss)
        except Exception as e:
            logging.error("Calculating perplexity failed with error: %s", e)
            perplexity = 100000

        if math.isnan(perplexity):
            perplexity = 100000

        self._last_epoch = epoch_idx

        self.context.backward(loss)
        self.context.step_optimizer(
            self._optimizer,
            clip_grads=lambda params: torch.nn.utils.clip_grad_norm_(
                params,
                self.context.get_hparam("clip_gradients_l2_norm"),
            ),
        )

        return {"loss": loss, "raw_loss": raw_loss, "perplexity": perplexity}

    def evaluate_full_dataset(self, data_loader: torch.utils.data.DataLoader):
        """
        Evaluates the full dataset against the given arch
        """
        # If optimizer is ASGD, we'll have to save current params
        # to a tmp var and copy over averaged params to use for eval.
        if self._optimizer.optim_name == "ASGD":
            tmp = {}
            for prm in self.model.parameters():
                tmp[prm] = prm.data.clone()
                prm.data = self._optimizer.ASGD.state[prm]["ax"].clone()

        hidden = self.model.init_hidden(self.hparams.eval_batch_size)

        total_loss = 0
        num_samples_seen = 0
        for i, batch in enumerate(data_loader):
            features, targets = batch
            features, targets = features.cuda(), targets.cuda()

            log_prob, hidden = self.model(features, hidden)
            loss = nn.functional.nll_loss(
                log_prob.contiguous().view(-1, log_prob.size(2)), targets
            ).data
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

        # Update eval history
        self._last_loss = total_loss
        best_loss = min(
            total_loss,
            float("inf") if not len(self._eval_history) else min(self._eval_history),
        )
        self._eval_history.append(best_loss)

        # If optimizer is ASGD, restore current params
        if self._optimizer.optim_name == "ASGD":
            for prm in self.model.parameters():
                prm.data = tmp[prm].clone()

        return {"loss": total_loss, "perplexity": perplexity}
