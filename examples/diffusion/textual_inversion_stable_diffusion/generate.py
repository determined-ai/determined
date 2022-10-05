import determined as det
import logging

from detsd import DetSDTextualInversionPipeline

logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)


if __name__ == "__main__":
    DetSDTextualInversionPipeline.generate_on_cluster()
