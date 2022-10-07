import determined as det
import logging

from detsd import DetSDTextualInversionTrainer

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    DetSDTextualInversionTrainer.train_on_cluster()
