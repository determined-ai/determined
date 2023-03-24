import logging

from detsd import DetSDTextualInversionTrainer

import determined as det

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    DetSDTextualInversionTrainer.train_on_cluster()
