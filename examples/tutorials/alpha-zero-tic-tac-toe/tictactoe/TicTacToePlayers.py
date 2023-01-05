import numpy as np

"""
Random and Human-ineracting players for the game of TicTacToe.

Author: Evgeny Tyurin, github.com/evg-tyurin
Date: Jan 5, 2018.

Based on the OthelloPlayers by Surag Nair.

"""


class RandomPlayer:
    def __init__(self, game):
        self.game = game

    def play(self, board):
        a = np.random.randint(self.game.getActionSize())
        valids = self.game.getValidMoves(board, 1)
        while valids[a] != 1:
            a = np.random.randint(self.game.getActionSize())
        return a


class HumanTicTacToePlayer:
    def __init__(self, game):
        self.game = game

    def play(self, board):
        # display(board)
        valid = self.game.getValidMoves(board, 1)
        for i in range(len(valid)):
            if valid[i]:
                print(int(i / self.game.n), int(i % self.game.n))
        while True:
            a = input()
            x, y = [int(x) for x in a.split(" ")]
            a = self.game.n * x + y if x != -1 else self.game.n ** 2
            if valid[a]:
                break
            else:
                print("Invalid")

        return a


"""
MinMax player for the game of TicTacToe.

Author: Erik Wilson
Date: Dec 15, 2022.

"""


class MinMaxPlayer:
    def __init__(self, game):
        self.game = game
        self.memoize = {}

    def play(self, board):
        return self.minMax(board, 0)[1]

    def minMax(self, board, depth):
        endGame = self.game.getGameEnded(board, 1)
        if abs(endGame) == 1:
            return (endGame * (self.game.getActionSize() - depth), None)
        elif endGame != 0:
            return (endGame, None)
        currentPlayer = 1 if depth % 2 == 0 else -1
        valids = self.game.getValidMoves(board, currentPlayer)
        candidates = []
        for a in range(self.game.getActionSize()):
            if valids[a] == 0:
                continue
            nextBoard, _nextPlayer = self.game.getNextState(board, currentPlayer, a)
            nextBoardKey = self.getBoardKey(nextBoard, currentPlayer)
            score = None
            if nextBoardKey in self.memoize:
                score = self.memoize[nextBoardKey]
            else:
                (score, _action) = self.minMax(nextBoard, depth + 1)
                self.memoize[nextBoardKey] = score
            candidates += [(score, a)]
        candidates.sort()
        candidate = candidates[(depth % 2) - 1]
        if depth != 0:
            return candidate
        targetScore = candidate[0]
        minRange = None
        maxRange = None
        for i, (score, _action) in enumerate(candidates):
            if score == targetScore:
                if minRange == None:
                    minRange = i
                maxRange = i
        if minRange == maxRange:
            return candidate
        candidates = candidates[minRange:maxRange]
        return candidates[np.random.randint(len(candidates))]

    def getBoardKey(self, board, player):
        l = []
        for i in range(1, 5):
            newB = np.rot90(board, i)
            l += [newB.tostring()]
            l += [np.fliplr(newB).tostring()]
        l.sort()
        return str(player) + str(l[-1])
