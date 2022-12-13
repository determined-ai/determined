import logging

from detsd import DetSDTextualInversionPipeline

import determined as det

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format=det.LOG_FORMAT)
    DetSDTextualInversionPipeline.generate_on_cluster()
