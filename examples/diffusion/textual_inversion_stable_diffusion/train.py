import determined as det
import logging

from detsd import DetSDTextualInversionTrainer

logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)


if __name__ == "__main__":
    DetSDTextualInversionTrainer.train_on_cluster()
