import logging

from Coach import Coach
from tictactoe.TicTacToeGame import TicTacToeGame as Game
from tictactoe.keras.NNet import NNetWrapper as nn
from utils import *

from DeterminedShim import shim

log = logging.getLogger(__name__)

args = dotdict(
    shim.override_params(
        {
            "numIters": shim.max_length(5),
            "numEps": 25,  # Number of complete self-play games to simulate during a new iteration.
            "tempThreshold": 15,  #
            "updateThreshold": 0.6,  # During arena playoff, new neural net will be accepted if threshold or more of games are won.
            "maxlenOfQueue": 200000,  # Number of game examples to train the neural networks.
            "numMCTSSims": 50,  # Number of games moves for MCTS to simulate.
            "arenaCompare": 50,  # Number of games to play during arena play to determine if new net will be accepted.
            "cpuct": 1,
            "checkpoint_path": "checkpoints",
            "checkpoint_file": "best.pth.tar",
            "numItersForTrainExamplesHistory": 20,
            "stop_after": 25,  # Number of minutes after which we should early exit when starting a new iteration
            "max_draws": 3,  # Maximum number of tied-game rounds before stopping
        }
    )
)


def main():
    log.info("{🎲} Loading %s...", Game.__name__)
    g = Game()

    log.info("{🧠} Loading %s...", nn.__name__)
    nnet = nn(g)

    (checkpoint_loaded, _metadata) = nnet.load_checkpoint(
        args.checkpoint_path, args.checkpoint_file, required=False
    )

    log.info("{⏰} Loading the Coach...")
    c = Coach(g, nnet, args)

    if checkpoint_loaded == True:
        log.info("{🏃} Loading training examples from file...")
        c.loadTrainExamples()

    log.info("{🎉} Starting the learning process")
    c.learn()
    log.info("{✅} All done!")


if __name__ == "__main__":
    main()
