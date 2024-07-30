from unittest import TextTestResult, TextTestRunner
from unittest.main import TestProgram
import sys

from . import venv
from ..models.session import session


class TestProgramWithConfig(TestProgram):
    def __init__(self):
        super().__init__(module=None, testRunner=Runner, exit=False)

    @staticmethod
    def pre():
        venv.make()
        venv.activate()
        session.start()
        print(f'Results: {session.result.dir()}')

    def runTests(self):
        self.pre()
        super().runTests()
        self.post()

    def post(self):
        session.stop()
        print(f'Results: {session.result.dir()}')
        sys.exit(not self.result.wasSuccessful())


class Runner(TextTestRunner):
    def __init__(self):
        super().__init__(resultclass=TestResult)


class TestResult(TextTestResult):
    pass


main = TestProgramWithConfig
