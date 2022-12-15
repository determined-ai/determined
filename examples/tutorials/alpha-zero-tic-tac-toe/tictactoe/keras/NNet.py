import logging
import os
import time
import numpy as np
import sys

sys.path.append("..")
from utils import *
from NeuralNet import NeuralNet
from DeterminedShim import shim

from .TicTacToeNNet import TicTacToeNNet as onnet

log = logging.getLogger(__name__)
mod = sys.modules[__name__]

"""
NeuralNet wrapper class for the TicTacToeNNet.

Author: Evgeny Tyurin, github.com/evg-tyurin
Date: Jan 5, 2018.

Based on (copy-pasted from) the NNet by SourKream and Surag Nair.
"""

args = dotdict(
    shim.override_params(
        {
            "lr": 0.001,
            "dropout": 0.3,
            "epochs": 10,
            "batch_size": 64,
            "num_channels": 512,
        },
        "checkpoints",
        "nnet_args",
    )
)


class NNetWrapper(NeuralNet):
    def __init__(self, game):
        self.nnet = onnet(game, args)
        self.board_x, self.board_y = game.getBoardSize()
        self.action_size = game.getActionSize()
        self.checkpoint_count = 0

    def train(self, examples):
        """
        examples: list of examples, each example is of form (board, pi, v)
        """
        input_boards, target_pis, target_vs = list(zip(*examples))
        input_boards = np.asarray(input_boards)
        target_pis = np.asarray(target_pis)
        target_vs = np.asarray(target_vs)
        return self.nnet.model.fit(
            x=input_boards,
            y=[target_pis, target_vs],
            batch_size=args.batch_size,
            epochs=args.epochs,
        )

    def predict(self, board):
        """
        board: np array with board
        """
        # timing
        start = time.time()

        # preparing input
        board = board[np.newaxis, :, :]

        # run
        pi, v = self.nnet.model.predict(board, verbose=False)

        # print('PREDICTION TIME TAKEN : {0:03f}'.format(time.time()-start))
        return pi[0], v[0]

    # shim for modifying path on save if Determined cluster is available
    def save_checkpoint(self, folder="checkpoints", filename="checkpoint.pth.tar"):
        folder = shim.save_path(folder)
        write_json(folder, "nnet_args", {"num_channels": args.num_channels})
        self.internal_save_checkpoint(folder, filename)

    def internal_save_checkpoint(self, folder="checkpoints", filename="checkpoint.pth.tar"):
        # change extension
        filename = filename.split(".")[0] + ".h5"

        filepath = os.path.join(folder, filename)

        if not os.path.exists(folder):
            log.info("Checkpoint Directory does not exist! Making directory {}".format(folder))
            os.mkdir(folder)

        log.info("Checkpoint saving to %s", filepath)
        self.nnet.model.save_weights(filepath)

    # shim for modifying path on load if Determined cluster is available
    def load_checkpoint(self, folder="checkpoints", filename="checkpoint.pth.tar", required=True):
        (folder, metadata) = shim.load_path(folder)
        return (self.internal_load_checkpoint(folder, filename, required), metadata)

    def internal_load_checkpoint(
        self, folder="checkpoints", filename="checkpoint.pth.tar", required=True
    ):
        # change extension
        filename = filename.split(".")[0] + ".h5"

        filepath = os.path.join(folder, filename)

        if not os.path.exists(filepath):
            log.warn("Checkpoint not available at %s", filepath)
            if required:
                raise ("No model in path")
            return False

        log.info("Checkpoint loading from %s", filepath)
        self.nnet.model.load_weights(filepath)
        return True
