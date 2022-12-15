import Arena
from MCTS import MCTS
from tictactoe.TicTacToeGame import TicTacToeGame as Game
from tictactoe.keras.NNet import NNetWrapper as NNet
from tictactoe.TicTacToePlayers import *

import numpy as np
from utils import *

import sys

"""
use this script to play any two agents against each other, or play manually with
any agent.
"""

g = Game()


def nn_player():
    nn = NNet(g)
    nn.load_checkpoint("checkpoints", "best.pth.tar")
    args = dotdict({"numMCTSSims": 50, "cpuct": 1.0})
    mcts = MCTS(g, nn, args)
    nnp = lambda x: np.argmax(mcts.getActionProb(x, temp=0))
    return nnp


def get_player(type):
    if type == "minmax":
        return MinMaxPlayer(g).play
    elif type == "random":
        return RandomPlayer(g).play
    elif type == "human":
        return HumanTicTacToePlayer(g).play
    elif type == "net":
        return nn_player()
    else:
        raise (Exception("unknown player type", type))


def get_arg(n, default):
    arg = sys.argv[n + 1 :]
    return arg[0] if arg else default


rounds = int(get_arg(0, "200"))
player1 = get_player(get_arg(1, "minmax"))
player2 = get_player(get_arg(2, "net"))

arena = Arena.Arena(player1, player2, g, display=Game.display)
print(arena.playGames(rounds, verbose=True))
