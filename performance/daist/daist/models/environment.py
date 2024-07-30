from pathlib import Path
from typing import Optional, NewType
import os

from .base import BaseDict
from ..framework.paths import ConfigDPath
from ..framework.typelib import Url_t


class Environment(BaseDict):
    class Key:
        """An enumeration of supported environment variables that will be serialized."""
        type_ = NewType('Env', str)

        DAIST_CONFIG: type_ = 'DAIST_CONFIG'
        DET_MASTER: type_ = 'DET_MASTER'
        DET_USER: type_ = 'DET_USER'
        GIT_BRANCH: type_ = 'GIT_BRANCH'
        GIT_COMMIT: type_ = 'GIT_COMMIT'
        PERF_RESULT_DB_HOST: type_ = 'PERF_RESULT_DB_HOST'
        PERF_RESULT_DB_USER: type_ = 'PERF_RESULT_DB_USER'

        all_ = (DAIST_CONFIG, DET_MASTER, DET_USER, GIT_BRANCH, GIT_COMMIT, PERF_RESULT_DB_HOST,
                PERF_RESULT_DB_USER)

    def __init__(self, dict_=None, /, **kwargs):
        super().__init__(dict_, **kwargs)

        if self.Key.DAIST_CONFIG not in self:
            self[self.Key.DAIST_CONFIG] = os.environ.get(self.Key.DAIST_CONFIG,
                                                         ConfigDPath.DEFAULT_CONFIG)
        if self.Key.DET_MASTER not in self:
            self[self.Key.DET_MASTER] = os.environ.get(self.Key.DET_MASTER)
        if self.Key.DET_USER not in self:
            self[self.Key.DET_USER] = os.environ.get(self.Key.DET_USER)
        if self.Key.GIT_BRANCH not in self:
            self[self.Key.GIT_BRANCH] = os.environ.get(self.Key.GIT_BRANCH)
        if self.Key.GIT_COMMIT not in self:
            self[self.Key.GIT_COMMIT] = os.environ.get(self.Key.GIT_COMMIT)
        if self.Key.PERF_RESULT_DB_HOST not in self:
            self[self.Key.PERF_RESULT_DB_HOST] = os.environ.get(self.Key.PERF_RESULT_DB_HOST)
        if self.Key.PERF_RESULT_DB_USER not in self:
            self[self.Key.PERF_RESULT_DB_USER] = os.environ.get(self.Key.PERF_RESULT_DB_USER)

        # This will not be serialized - it is to remain in RAM only.
        self.secrets = SecretEnvironment()

    @property
    def daist_config(self) -> Path:
        return self[self.Key.DAIST_CONFIG]

    @property
    def det_master(self) -> Optional[Url_t]:
        return self[self.Key.DET_MASTER]

    @property
    def det_user(self) -> Optional[str]:
        return self[self.Key.DET_USER]

    @property
    def git_branch(self) -> Optional[str]:
        return self[self.Key.GIT_BRANCH]

    @property
    def git_commit(self) -> Optional[str]:
        return self[self.Key.GIT_COMMIT]

    @property
    def perf_result_db_host(self) -> Optional[str]:
        return self[self.Key.PERF_RESULT_DB_HOST]

    @property
    def perf_result_db_user(self) -> Optional[str]:
        return self[self.Key.PERF_RESULT_DB_USER]

    @classmethod
    def _deserialize(cls, key, value):
        if key == cls.Key.DAIST_CONFIG and value is not None:
            value = Path(value)
        return super()._deserialize(key, value)


class SecretEnvironment(BaseDict):
    class Key:
        type_ = NewType('Environment.SecretKey', str)
        DET_PASSWORD: type_ = 'DET_PASSWORD'
        PERF_RESULT_DB_PASS: type_ = 'PERF_RESULT_DB_PASS'
        PGPASSWORD: type_ = 'PGPASSWORD'

        all_ = (DET_PASSWORD, PERF_RESULT_DB_PASS, PGPASSWORD)

    def __init__(self, dict_=None, /, **kw):
        super().__init__(dict_, **kw)

        for key in self.Key.all_:
            self[key] = os.environ.get(key)

    @property
    def det_pass(self) -> Optional[str]:
        return self[self.Key.DET_PASSWORD]

    @property
    def perf_result_db_pass(self) -> Optional[str]:
        """
        Order of precedence:

        - The database specific environment variable
        - The base postgres environment variable
        """
        ret = self[self.Key.PERF_RESULT_DB_PASS]
        if ret is None:
            ret = self[self.Key.PGPASSWORD]
        return ret

    @property
    def pgpassword(self) -> Optional[str]:
        return self[self.Key.PGPASSWORD]


#: Application global singleton
environment = Environment()
