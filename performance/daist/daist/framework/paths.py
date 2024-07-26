from pathlib import Path
import os


class RootPath:
    PATH = Path(os.path.relpath(Path(__file__).parent.parent.parent.resolve(), os.getcwd()))
    BUILD_OUTPUT = PATH / '__build__'
    DIST_OUTPUT = PATH / '__dist__'
    SETUP_PY = PATH / 'setup.py'
    PKG = PATH / 'daist'
    VENV = PATH / '__venv__'


class PkgPath:
    PATH = RootPath.PKG
    CONFIG_D = PATH / 'config.d'
    DEFAULT_RESULTS = PATH / 'results'


class ConfigDPath:
    DEFAULT_CONFIG = PkgPath.CONFIG_D / 'config.conf'


class VenvPath:
    PATH = RootPath.VENV
    PIP_HISTORY = PATH / 'pip_history.pkl'
    BIN = PATH / 'bin'
    DET = BIN / 'det'
    PIP = BIN / 'pip'
    PYTHON = BIN / 'python'
