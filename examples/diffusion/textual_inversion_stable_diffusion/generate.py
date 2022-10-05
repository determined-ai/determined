import determined as det
import logging

from detsd import DetSDTextualInversionPipeline

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    DetSDTextualInversionPipeline.generate_on_cluster()
