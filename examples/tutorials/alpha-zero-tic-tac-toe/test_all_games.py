""""

    This is a Regression Test Suite to automatically test all combinations of games and ML frameworks. Each test
    plays two quick games using an untrained neural network (randomly initialized) against a random player.

    In order for the entire test suite to run successfully, all the required libraries must be installed.  They are:
    Pytorch, Keras.

     [ Games ]      Pytorch      Keras
      -----------   -------      -----
    - TicTacToe                  [Yes]
"""

import unittest

import Arena
from MCTS import MCTS

from tictactoe.TicTacToeGame import TicTacToeGame
from tictactoe.TicTacToePlayers import RandomPlayer
from tictactoe.keras.NNet import NNetWrapper as TicTacToeKerasNNet

import numpy as np
from utils import *

class TestAllGames(unittest.TestCase):

    @staticmethod
    def execute_game_test(game, neural_net):
        rp = RandomPlayer(game).play

        args = dotdict({'numMCTSSims': 25, 'cpuct': 1.0})
        mcts = MCTS(game, neural_net(game), args)
        n1p = lambda x: np.argmax(mcts.getActionProb(x, temp=0))

        arena = Arena.Arena(n1p, rp, game)
        print(arena.playGames(2, verbose=False))
   
    def test_tictactoe_keras(self):
        self.execute_game_test(TicTacToeGame(), TicTacToeKerasNNet)

if __name__ == '__main__':
    unittest.main()
