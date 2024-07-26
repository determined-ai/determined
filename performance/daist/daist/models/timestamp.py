from datetime import datetime
from typing import NewType, Optional

from .base import BaseDict
from ..utils import timestamp


class UnixTime(BaseDict):
    class Key:
        type_ = NewType('Time.Key', str)
        UNIX: type_ = 'unix'

    def __init__(self, ts: Optional[timestamp.TS_t] = None):
        super().__init__()
        self.unix = ts

    @property
    def dt(self) -> datetime:
        return timestamp.get_utc_dt(self.unix)

    @dt.setter
    def dt(self, ts: timestamp.TS_t):
        self.unix = ts

    @property
    def unix(self) -> int:
        return self[self.Key.UNIX]

    @unix.setter
    def unix(self, ts: timestamp.TS_t):
        self[self.Key.UNIX] = ts

    @classmethod
    def _serialize(cls, key, value: timestamp.TS_t):
        if key == cls.Key.UNIX:
            value = int(timestamp.get_unix_time(value))
        return super()._serialize(key, value)


class UnixTimeWithStamp(UnixTime):
    FMT_FOR_PATH = f'%Y-%m-%dT%H-%M-%S{timestamp.UTC_Z}'

    class Key(UnixTime.Key):
        STAMP: UnixTime.Key.type_ = 'stamp'

    @property
    def stamp(self) -> str:
        return self[self.Key.STAMP]

    @stamp.setter
    def stamp(self, value: timestamp.TS_t):
        self[self.Key.STAMP] = value

    @property
    def path_stamp(self) -> str:
        return self.dt.strftime(self.FMT_FOR_PATH)

    @classmethod
    def _serialize(cls, key, value):
        if key == cls.Key.STAMP:
            value = timestamp.get_utc_iso_ts_str(value, timespec='seconds')
        return super()._serialize(key, value)

