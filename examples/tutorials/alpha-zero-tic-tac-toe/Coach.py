import logging
import os
from collections import deque
from pickle import Pickler, Unpickler
from datetime import datetime
from random import shuffle

import numpy as np
from tqdm import tqdm

from Arena import Arena
from MCTS import MCTS

from DeterminedShim import shim

log = logging.getLogger(__name__)


class Coach():
    """
    This class executes the self-play + learning. It uses the functions defined
    in Game and NeuralNet. args are specified in main.py.
    """

    def __init__(self, game, nnet, args):
        self.game = game
        self.nnet = nnet
        self.pnet = self.nnet.__class__(self.game)  # the competitor network
        self.args = args
        self.mcts = MCTS(self.game, self.nnet, self.args)
        self.trainExamplesHistory = []  # history of examples from args.numItersForTrainExamplesHistory latest iterations
        self.skipFirstSelfPlay = False  # can be overriden in loadTrainExamples()

    def executeEpisode(self):
        """
        This function executes one episode of self-play, starting with player 1.
        As the game is played, each turn is added as a training example to
        trainExamples. The game is played till the game ends. After the game
        ends, the outcome of the game is used to assign values to each example
        in trainExamples.

        It uses a temp=1 if episodeStep < tempThreshold, and thereafter
        uses temp=0.

        Returns:
            trainExamples: a list of examples of the form (canonicalBoard, currPlayer, pi,v)
                           pi is the MCTS informed policy vector, v is +1 if
                           the player eventually won the game, else -1.
        """
        trainExamples = []
        board = self.game.getInitBoard()
        self.curPlayer = 1
        episodeStep = 0

        while True:
            episodeStep += 1
            canonicalBoard = self.game.getCanonicalForm(board, self.curPlayer)
            temp = int(episodeStep < self.args.tempThreshold)

            pi = self.mcts.getActionProb(canonicalBoard, temp=temp)
            sym = self.game.getSymmetries(canonicalBoard, pi)
            for b, p in sym:
                trainExamples.append([b, self.curPlayer, p, None])

            action = np.random.choice(len(pi), p=pi)
            board, self.curPlayer = self.game.getNextState(board, self.curPlayer, action)

            r = self.game.getGameEnded(board, self.curPlayer)

            if r != 0:
                return [(x[0], x[2], r * ((-1) ** (x[1] != self.curPlayer))) for x in trainExamples]

    def learn(self):
        """
        Performs numIters iterations with numEps episodes of self-play in each
        iteration. After every iteration, it retrains neural network with
        examples in trainExamples (which has a maximum length of maxlenofQueue).
        It then pits the new neural network against the old one and accepts it
        only if it wins >= updateThreshold fraction of games.
        """
        all_draws = 0
        learn_start = datetime.now()

        for i in range(1, self.args.numIters + 1):
            # initialize Determined shim for this step
            shim.step()

            iter_start = datetime.now()

            # bookkeeping
            log.info(f'Starting Iter #{i} ...')

            # examples of the iteration
            if not self.skipFirstSelfPlay or i > 1:
                iterationTrainExamples = deque([], maxlen=self.args.maxlenOfQueue)

                for _ in tqdm(range(self.args.numEps), desc="Self Play"):
                    self.mcts = MCTS(self.game, self.nnet, self.args)  # reset search tree
                    iterationTrainExamples += self.executeEpisode()

                # save the iteration examples to the history 
                self.trainExamplesHistory.append(iterationTrainExamples)

            if len(self.trainExamplesHistory) > self.args.numItersForTrainExamplesHistory:
                log.warning(
                    f"Removing the oldest entry in trainExamples. len(trainExamplesHistory) = {len(self.trainExamplesHistory)}")
                self.trainExamplesHistory.pop(0)
            # backup history to a file
            # NB! the examples were collected using the model from the previous iteration, so (i-1)  
            self.saveTrainExamples()

            # shuffle examples before training
            trainExamples = []
            for e in self.trainExamplesHistory:
                trainExamples.extend(e)
            shuffle(trainExamples)

            # training new network, keeping a copy of the old one
            self.nnet.internal_save_checkpoint(folder=self.args.checkpoint_path, filename='temp.pth.tar')
            self.pnet.internal_load_checkpoint(folder=self.args.checkpoint_path, filename='temp.pth.tar')

            pmcts = MCTS(self.game, self.pnet, self.args)

            shim.training_metrics(self.nnet.train(trainExamples).history)

            nmcts = MCTS(self.game, self.nnet, self.args)

            log.info('PITTING AGAINST PREVIOUS VERSION')
            arena = Arena(lambda x: np.argmax(pmcts.getActionProb(x, temp=0)),
                          lambda x: np.argmax(nmcts.getActionProb(x, temp=0)), self.game)
            pwins, nwins, draws = arena.playGames(self.args.arenaCompare)

            log.info('NEW/PREV WINS : %d / %d ; DRAWS : %d' % (nwins, pwins, draws))

            if pwins + nwins == 0 or float(nwins) / (pwins + nwins) < self.args.updateThreshold:
                log.info('REJECTING NEW MODEL')
                self.nnet.internal_load_checkpoint(folder=self.args.checkpoint_path, filename='temp.pth.tar')
            else:
                log.info('ACCEPTING NEW MODEL')

            self.nnet.save_checkpoint(folder=self.args.checkpoint_path, filename='best.pth.tar')

            # add extra metrics to help with determined hyper parameter searching:
            # end training early when the matches results in no wins for some consecutive iterations
            if pwins == nwins == 0:
                all_draws += 1
            else:
                all_draws = 0

            iter_dt = (datetime.now() - iter_start).total_seconds() / 60.0
            learn_dt = (datetime.now() - learn_start).total_seconds() / 60.0

            validation_metrics = {
                'nwins': nwins,
                'pwins': pwins,
                'draws': draws,
                'iter_dt': iter_dt,
                'iter_draws_per_minute': draws/iter_dt,
                'learn_dt': learn_dt,
            }

            # define a metric for validation where we have a parameter that allows us to find
            # the fastest possible time to solution
            if all_draws >= self.args['max_draws']:
                validation_metrics['max_draws_per_minute'] = draws/learn_dt

            shim.validation_metrics(validation_metrics)

            if all_draws >= self.args['max_draws']:
                log.info('NO WINNERS, ENDING PLAY')
                break

            if learn_dt > self.args.stop_after:
                log.warn('OUT OF TIME, STOPPING EARLY')
                break

    # shim for modifying path on save if Determined cluster is available
    def saveTrainExamples(self):
        self.internal_saveTrainExamples(shim.save_path(self.args.checkpoint_path))

    def internal_saveTrainExamples(self, folder):
        if not os.path.exists(folder):
            os.makedirs(folder)

        filename = os.path.join(folder, "latest.examples")
        with open(filename, "wb+") as f:
            Pickler(f).dump(self.trainExamplesHistory)

    # shim for modifying path on load if Determined cluster is available
    def loadTrainExamples(self, required=True):
        (folder, metadata) = shim.load_path(self.args.checkpoint_path)        
        return (self.internal_loadTrainExamples(folder, required), metadata)

    def internal_loadTrainExamples(self, folder, required=True):
        filename = os.path.join(folder, "latest.examples")

        if not os.path.isfile(filename):
            log.warning(f'File "{filename}" with training examples not found!')
            if required:
                raise("No examples in path")
            return False

        with open(filename, "rb") as f:
            self.trainExamplesHistory = Unpickler(f).load()

        # examples based on the model were already collected (loaded)
        self.skipFirstSelfPlay = True
        return True
