from packaging.version import Version as PyVersion
from pathlib import Path
from typing import Iterable, Mapping, MutableMapping, NewType, Optional, Union
from urllib.parse import urlparse
import shutil
import shlex
import sys

from .base import BaseDict, BaseObj, Format
from .environment import Environment
from .timestamp import UnixTime, UnixTimeWithStamp
from ..framework.paths import PkgPath
from ..framework.typelib import TestId_t, Url_t


class DeterminedVersion(BaseDict):
    class Key:
        CLIENT = 'client'
        SERVER = 'server'

    @property
    def client(self) -> PyVersion:
        return self[self.Key.CLIENT]

    @client.setter
    def client(self, value: Union[str, PyVersion]):
        self[self.Key.CLIENT] = value

    @property
    def server(self) -> PyVersion:
        return self[self.Key.SERVER]

    @server.setter
    def server(self, value: Union[str, PyVersion]):
        self[self.Key.SERVER] = value

    @staticmethod
    def _deserialize(key, value):
        return BaseDict._deserialize(key, PyVersion(value))

    @classmethod
    def _serialize(cls, key, value):
        return super()._serialize(key, cls._serialize_to_str(value))


class Host(BaseDict):
    class Key:
        DETERMINED = 'determined'

    @property
    def determined(self) -> Url_t:
        return self[self.Key.DETERMINED]

    @determined.setter
    def determined(self, value: Url_t):
        self[self.Key.DETERMINED] = value


class Result(BaseDict):

    @staticmethod
    def _adding_to(method):
        def wrap(*args, meta: Optional['FileMeta'] = None):
            self: Result = args[0]
            src: Union[None, str, Path, BaseObj]
            rel_dst: Union[str, Path]

            try:
                src = args[1]
                rel_dst = args[2]
            except IndexError:
                src = None
                rel_dst = args[1]

            # Type-hinting and casting
            rel_dst = Path(rel_dst)

            # Destination directory resolution
            if rel_dst.is_dir():
                if isinstance(src, BaseObj):
                    dst = self.dir() / rel_dst / src.get_filename()
                else:
                    dst = self.dir() / rel_dst / src.name
            else:
                dst = self.dir() / rel_dst

            # Destination directory prep
            dst.parent.mkdir(parents=True, exist_ok=True)

            # Call the wrapped method
            if src is None:
                ret = method(self, dst)
            else:
                ret = method(self, src, dst)

            # Create meta-data as necessary
            if meta is None:
                meta = FileMeta()

            # Cache map the metadata to the destination path.
            self.files[dst.relative_to(self.dir())] = meta

            return ret

        return wrap

    class Key:
        type_ = NewType('Results.Key', str)
        CLASSNAME: type_ = 'classname'
        COMMAND: type_ = 'command'
        HOST: type_ = 'host'
        END_TIME: type_ = 'end_time'
        ENVIRONMENT: type_ = 'environment'
        FILES: type_ = 'files'
        START_TIME: type_ = 'start_time'
        VERSION: type_ = 'version'

    def __init__(self, dict_=None, /, **kwargs):
        super().__init__(dict_, **kwargs)

        if self.Key.CLASSNAME not in self:
            self[self.Key.CLASSNAME] = self.get_qualname()

        if self.Key.COMMAND not in self:
            self[self.Key.COMMAND] = shlex.join(sys.argv[1:])

        if self.Key.FILES not in self:
            self[self.Key.FILES] = Files()

        if self.Key.HOST not in self:
            self[self.Key.HOST] = Host()

        if self.Key.START_TIME not in self:
            self[self.Key.START_TIME] = UnixTimeWithStamp()

        if self.Key.VERSION not in self:
            self[self.Key.VERSION] = Version()

    @property
    def classname(self) -> str:
        return self[self.Key.CLASSNAME]

    @classname.setter
    def classname(self, value: str):
        self[self.Key.CLASSNAME] = value

    @property
    def command(self) -> str:
        return self[self.Key.COMMAND]

    @command.setter
    def command(self, value: str):
        self[self.Key.COMMAND] = value

    @property
    def end_time(self) -> 'UnixTimeWithStamp':
        return self[self.Key.END_TIME]

    @end_time.setter
    def end_time(self, value: Mapping):
        self[self.Key.END_TIME] = value

    @property
    def environment(self) -> Environment:
        return self[self.Key.ENVIRONMENT]

    @environment.setter
    def environment(self, value: Mapping):
        self[self.Key.ENVIRONMENT] = value

    @property
    def files(self) -> 'Files':
        return self[self.Key.FILES]

    @files.setter
    def files(self, value: MutableMapping):
        self[self.Key.FILES] = value

    @property
    def host(self) -> Host:
        return self[self.Key.HOST]

    @host.setter
    def host(self, value: Union[dict, Host]):
        self[self.Key.HOST] = value

    @property
    def start_time(self) -> 'UnixTimeWithStamp':
        return self[self.Key.START_TIME]

    @start_time.setter
    def start_time(self, value: Mapping):
        self[self.Key.START_TIME] = value

    @property
    def version(self) -> 'Version':
        return self[self.Key.VERSION]

    @version.setter
    def version(self, value: Union[dict, 'Version']):
        self[self.Key.VERSION] = value

    # noinspection PyUnusedLocal
    @_adding_to
    def add_obj(self, src: BaseObj, dst: Union[str, Path],
                meta: Optional['FileMeta'] = None):
        src.save(dst)

    # noinspection PyUnusedLocal
    @_adding_to
    def copyfile(self, src: Union[str, Path], dst: Union[str, Path],
                 meta: Optional['FileMeta'] = None):
        shutil.copyfile(src, dst)

    # noinspection PyUnusedLocal
    @_adding_to
    def mv(self, src: Union[str, Path], dst: Union[str, Path],
           meta: Optional['FileMeta'] = None):
        shutil.move(src, dst)

    def dir(self) -> Path:
        return self._path.parent

    def get_filename(self, fmt: Format.type_ = Format.JSON,
                     tags: Optional[Iterable[str]] = None) -> Path:
        required_tags = [urlparse(self.host.determined).hostname, self.start_time.path_stamp]
        if tags is None:
            tags = required_tags
        else:
            tags = required_tags.extend(tags)
        return super().get_filename(fmt, tags)

    def path(self) -> Path:
        return self._path

    # noinspection PyUnusedLocal
    @_adding_to
    def touch(self, path: Union[str, Path], meta: Optional['FileMeta'] = None) -> Path:
        path.touch(exist_ok=True)
        return path

    @classmethod
    def _deserialize(cls, key, value):
        if key in (cls.Key.END_TIME, cls.Key.START_TIME):
            value = UnixTimeWithStamp(value)
        elif key == cls.Key.ENVIRONMENT:
            value = Environment(value)
        elif key == cls.Key.FILES:
            value = Files(value)
        elif key == cls.Key.VERSION:
            value = Version(value)
        elif key == cls.Key.HOST:
            value = Host(value)
        return super()._deserialize(key, value)

    @classmethod
    def _serialize(cls, key, value):
        if key in (cls.Key.END_TIME, cls.Key.ENVIRONMENT, cls.Key.FILES, cls.Key.START_TIME,
                   cls.Key.VERSION, cls.Key.HOST):
            value = cls._serialize_to_dict(value)
        return super()._serialize(key, value)


class FileMeta(BaseDict):
    class Key:
        type_ = NewType('File.Key', str)
        CLASSNAME: type_ = 'classname'
        TEST_ID: type_ = 'test_id'
        TIME: type_ = 'time'

    def __init__(self, dict_=None, /, **kwargs):
        super().__init__(dict_, **kwargs)

        if self.Key.TIME not in self:
            self.time = UnixTime()

    @property
    def test_id(self) -> Optional[TestId_t]:
        return self.get(self.Key.TEST_ID)

    @test_id.setter
    def test_id(self, value: Optional[TestId_t]):
        self[self.Key.TEST_ID] = value

    @property
    def time(self) -> UnixTime:
        return self[self.Key.TIME]

    @time.setter
    def time(self, value: Mapping):
        self[self.Key.TIME] = value

    @classmethod
    def _deserialize(cls, key, value):
        if key == cls.Key.TIME:
            value = UnixTime(value)
        return super()._deserialize(key, value)

    @classmethod
    def _serialize(cls, key, value):
        if key == cls.Key.TIME:
            value = cls._serialize_to_dict(value)
        return super()._serialize(key, value)


class Files(BaseDict):

    @staticmethod
    def _deserialize(key, value):
        return FileMeta(value)

    @classmethod
    def _serialize(cls, key, value):
        return super()._serialize(key, cls._serialize_to_dict(value))
    
    def __setitem__(self, key, value):
        super().__setitem__(str(key), value)

    def get(self, key, default=None):
        return super().get(str(key), default)


class Version(BaseDict):
    class Key:
        DETERMINED = 'determined'
        DAIST = PkgPath.PATH.name

    def __init__(self, dict_=None, /, **kw):
        super().__init__(dict_, **kw)

        if self.Key.DETERMINED not in self:
            self[self.Key.DETERMINED] = DeterminedVersion()

    @property
    def determined(self) -> 'DeterminedVersion':
        return self[self.Key.DETERMINED]

    @determined.setter
    def determined(self, value: Union[dict, DeterminedVersion]):
        self[self.Key.DETERMINED] = value

    @property
    def daist(self) -> PyVersion:
        return self[self.Key.DAIST]

    @daist.setter
    def daist(self, value: Union[str, PyVersion]):
        self[self.Key.DAIST] = value

    @classmethod
    def _deserialize(cls, key, value):
        if key == cls.Key.DAIST:
            return PyVersion(value)
        elif key == cls.Key.DETERMINED:
            return DeterminedVersion(value)
        return super()._deserialize(key, value)

    @classmethod
    def _serialize(cls, key, value):
        if key == cls.Key.DAIST:
            value = cls._serialize_to_str(value)
        elif key == cls.Key.DETERMINED:
            value = cls._serialize_to_dict(value)
        return super()._serialize(key, value)
