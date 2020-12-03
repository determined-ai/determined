"""
This file loads the training and validation data for model_def
"""
import logging
import os
import shutil
import tarfile

import numpy as np
import torch
import wget

from torch.utils.data import Dataset


class PadSequence:
    def __call__(self, batch):
        features = batch[:-1]
        labels = batch[1:]

        features = torch.stack(features)
        labels = torch.stack(labels).contiguous().view(-1)

        return features, labels


class BatchSamp:
    def __init__(self, dataset, bptt, max_seq_length_delta, valid=False):
        self.valid = valid
        self.data_length = len(dataset) - 1 - 1
        self.bptt = bptt
        self.max_seq_length_delta = max_seq_length_delta

    def _calculate_seq_len(self, i):
        bptt = self.bptt if np.random.random() < 0.95 else self.bptt / 2.0
        seq_len = max(5, int(np.random.normal(bptt, 5)))
        seq_len = min(seq_len, self.bptt + self.max_seq_length_delta)
        seq_len = min(self.bptt if self.valid else seq_len, self.data_length - 1 - i)
        return seq_len

    def __len__(self):
        return self.data_length // self.bptt

    def __iter__(self):
        seq_len = 0 if not self.valid else self.bptt
        i = 0
        while i < self.data_length:
            seq_len = self._calculate_seq_len(i)
            start = i
            end = i + seq_len
            # sometimes the seq_len is 0
            # this means we have reached the end of the data
            if seq_len == 0:
                break
            yield list(range(start, end + 1))
            i += seq_len


class PTBData(Dataset):
    def __init__(self, data, batch_size, bptt, max_seq_length_delta, valid=False):
        self.batch_size = batch_size
        self.data = self.batchify(data)
        self.avg_seq_len = bptt
        self.max_seq_len = bptt + max_seq_length_delta
        self.valid = valid

    def batchify(self, data):
        nbatch = data.size(0) // self.batch_size
        data = data.narrow(0, 0, nbatch * self.batch_size)
        data = data.contiguous().view(self.batch_size, -1).t().contiguous()  # returns [29049, 32]
        return data

    def __len__(self):
        return len(self.data)

    def __getitem__(self, i):
        return self.data[i]


def download_data(data_loc):
    if os.path.exists(os.path.join(data_loc, "train.txt")) and os.path.exists(
        os.path.join(data_loc, "valid.txt")
    ):
        # Exit if the data already exists
        return data_loc

    if not os.path.isdir(data_loc):
        os.makedirs(data_loc)
    logging.info("downloading and extracting ...")

    url = "http://www.fit.vutbr.cz/~imikolov/rnnlm/simple-examples.tgz"
    data_file = "simple-examples.tgz"

    wget.download(url, data_loc + data_file)

    tf = tarfile.open(data_loc + data_file)
    tf.extractall(path=data_loc)
    tf.close()

    temp_data_dir = os.path.join(data_loc, "simple-examples/data")
    shutil.move(
        os.path.join(temp_data_dir, "ptb.train.txt"),
        os.path.join(data_loc, "train.txt"),
    )
    shutil.move(
        os.path.join(temp_data_dir, "ptb.valid.txt"),
        os.path.join(data_loc, "valid.txt"),
    )
    shutil.move(
        os.path.join(temp_data_dir, "ptb.test.txt"), os.path.join(data_loc, "test.txt")
    )

    logging.info("\tcompleted")
