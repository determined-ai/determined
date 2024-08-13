from pathlib import Path
from typing import MutableMapping, NewType, Optional, Union

from ..framework.typelib import Url_t
from .base import BaseDict


class Config(BaseDict):
    """The test framework configuration."""
    class Key:
        type_ = NewType('Config.Key', str)
        DETERMINED: type_ = 'determined'
        EXEC: type_ = 'exec'
        LOG: type_ = 'log'

        all = (DETERMINED, EXEC, LOG)

    @property
    def determined(self) -> 'Determined':
        return self[self.Key.DETERMINED]

    @determined.setter
    def determined(self, value: MutableMapping):
        self[self.Key.DETERMINED] = value

    @property
    def exec(self) -> 'Exec':
        return self[self.Key.EXEC]

    @exec.setter
    def exec(self, value: MutableMapping):
        self[self.Key.EXEC] = value

    @property
    def log(self) -> 'Log':
        return self[self.Key.LOG]

    @log.setter
    def log(self, value: MutableMapping):
        self[self.Key.LOG] = value

    def path(self) -> Path:
        return self._path

    @classmethod
    def _deserialize(cls, key, value):
        if key == cls.Key.DETERMINED:
            value = Determined(value)
        elif key == cls.Key.EXEC:
            value = Exec(value)
        elif key == cls.Key.LOG:
            value = Log(value)
        return super()._deserialize(key, value)

    @classmethod
    def _serialize(cls, key, value):
        return super()._serialize(key, cls._serialize_to_dict(value))


class Determined(BaseDict):
    DEFAULT_DET_MASTER = 'http://localhost:8080'

    class Key:
        type_ = NewType('Environment.Key', str)
        DET_MASTER: type_ = 'DET_MASTER'
        DET_PASS: type_ = 'DET_PASS'
        DET_USER: type_ = 'DET_USER'
        RESOURCE_PROFILE: type_ = 'RESOURCE_PROFILE'

        all = (DET_MASTER, DET_PASS, DET_USER, RESOURCE_PROFILE)
        defaults = {
            DET_PASS: '',
            DET_USER: 'admin',
            RESOURCE_PROFILE: 'autodiscover',
        }

    def __init__(self, dict_=None, /, **kwargs):
        # Clean the input
        dict_[self.Key.DET_MASTER] = dict_.get(self.Key.DET_MASTER,
                                               self.DEFAULT_DET_MASTER).strip().rstrip('/')

        super().__init__(dict_, **kwargs)

    @property
    def det_master(self) -> Url_t:
        return self[self.Key.DET_MASTER]

    @det_master.setter
    def det_master(self, value: str):
        self[self.Key.DET_MASTER] = value

    @property
    def det_pass(self) -> str:
        return self.get(self.Key.DET_PASS, self.Key.defaults[self.Key.DET_PASS])

    @det_pass.setter
    def det_pass(self, value: str):
        self[self.Key.DET_PASS] = value

    @property
    def det_user(self) -> str:
        return self.get(self.Key.DET_USER, self.Key.defaults[self.Key.DET_USER])

    @det_user.setter
    def det_user(self, value: str):
        self[self.Key.DET_USER] = value

    @property
    def resource_profile(self) -> str:
        return self.get(self.Key.RESOURCE_PROFILE, self.Key.defaults[self.Key.RESOURCE_PROFILE])

    @resource_profile.setter
    def resource_profile(self, value: str):
        self[self.Key.RESOURCE_PROFILE] = value


class Exec(BaseDict):
    class Key:
        type_ = NewType('Exec.Key', str)
        OUTPUT: type_ = 'output'

    @property
    def output(self) -> Optional[Path]:
        return self.get(self.Key.OUTPUT)

    @output.setter
    def output(self, value: Optional[Union[str, Path]]):
        self[self.Key.OUTPUT] = value

    @classmethod
    def _deserialize(cls, key, value):
        if key == cls.Key.OUTPUT:
            if value:
                value = Path(value)
            else:
                value = None
        return super()._deserialize(key, value)

    @classmethod
    def _serialize(cls, key, value):
        if key == cls.Key.OUTPUT:
            value = str(value)
        super()._serialize(key, value)


class Log(BaseDict):
    class Key:
        type_ = NewType('Log.Key', str)
        FILE_LEVEL: type_ = 'file_level'
        STDOUT_LEVEL: type_ = 'stdout_level'

    @property
    def file_level(self) -> Optional[str]:
        return self.get(self.Key.FILE_LEVEL)

    @file_level.setter
    def file_level(self, value: str):
        self[self.Key.FILE_LEVEL] = value

    @property
    def stdout_level(self) -> Optional[str]:
        return self.get(self.Key.STDOUT_LEVEL)

    @stdout_level.setter
    def stdout_level(self, value: str):
        self[self.Key.STDOUT_LEVEL] = value
