import logging

from determined.common.api import bindings


class SearchRunner:
    def __init__(self, search_method):
        self.search_method = search_method

    def run(self):
        logging.info("SearchRunner.run")

        while True:
            bindings.get_GetSearcherEvents()
