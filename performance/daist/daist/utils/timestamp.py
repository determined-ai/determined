from datetime import datetime, timezone
from typing import Optional, Union
import time

UTC_Z = 'Z'

BASE_FMT = '%Y-%m-%dT%H:%M:%S'

#: The timestamp string format.
FMT = f'{BASE_FMT}.%f{UTC_Z}'

PRECISION = 6

#: Return with precision 6 to match the datetime %f string formatter (6 decimal places)
#: See: https://docs.python.org/3/library/datetime.html#strftime-and-strptime-format-codes
TS_PRECISION_FMT = f'{{:.0{PRECISION}f}}'

TS_t = Union[str, datetime, int, float]


def get_unix_time(ts: Optional[TS_t] = None, fmt: str = FMT) -> float:
    """
    :param ts: A timestamp that may take the following forms:
        - a string, matching the :data:`FMT` format.
        - an integer or a float, in which case it represents the number of seconds since the unix
          epoch.
        - A datetime object. This object will be made timezone aware and set to UTC prior to
            returning the number of seconds since the unix epoch.
            Warning: timezone offsets will be applied if the input datetime object is naive and
            created outside the UTC timezone OR is timezone aware and not set to UTC.
        - None. In which case a timestamp will be created.
    :param fmt: The string parsing format.

    :return: The number of seconds since the Unix epoch as a floating point value using the
        format given by :data:`TS_PRECISION_FMT`.
    """
    if isinstance(ts, datetime):
        return ts.astimezone(timezone.utc).timestamp()
    elif isinstance(ts, (int, float, str)):
        try:
            return float(TS_PRECISION_FMT.format(float(ts)))
        except ValueError:
            try:
                return datetime.strptime(ts, fmt).replace(tzinfo=timezone.utc).timestamp()
            except ValueError:
                raise ValueError(f'Unable to convert "{ts}" to a UTC epoch timestamp.')
    else:
        return round(time.time(), PRECISION)


def get_utc_dt(ts: Optional[TS_t] = None, fmt: str = FMT) -> datetime:
    """
    :param ts: A timestamp that may take the following forms:
        - a string, matching the :data:`FMT` format.
        - an integer or a float, in which case it represents the number of seconds since the unix
          epoch.
        - A datetime object. This object will be made timezone aware and set to UTC prior to
            returning the number of seconds since the unix epoch.
            Warning: timezone offsets will be applied if the input datetime object is naive and
            created outside the UTC timezone OR is timezone aware and not set to UTC.
        - None. In which case a timestamp will be created.
    :param fmt: The string parsing format.

    :return: A timezone aware UTC datetime.
    """
    if isinstance(ts, datetime):
        dt = ts.astimezone(timezone.utc)
    elif isinstance(ts, str):
        dt = datetime.strptime(ts, fmt).replace(tzinfo=timezone.utc)
    elif isinstance(ts, (int, float)):
        dt = datetime.fromtimestamp(ts, tz=timezone.utc)
    else:
        dt = datetime.fromtimestamp(time.time(), tz=timezone.utc)
    return dt


def get_utc_iso_ts_str(ts: Optional[TS_t] = None,
                       fmt: str = FMT, timespec: str = 'microseconds') -> str:
    """
    :param ts: A timestamp that may take the following forms:
        - a string, matching the :data:`FMT` format.
        - an integer or a float, in which case it represents the number of seconds since the unix
          epoch.
        - A datetime object. This object will be made timezone aware and set to UTC prior to
            returning the number of seconds since the unix epoch.
            Warning: timezone offsets will be applied if the input datetime object is naive and
            created outside the UTC timezone OR is timezone aware and not set to UTC.
        - None. In which case a timestamp will be created.
    :param fmt: The string parsing format.
    :param timespec: The precision to format the string with.

    :return: A UTC timestamp string taking the form :data:`FMT`. The seconds will be given as a
        float with microsecond precision.
    """
    return get_utc_dt(ts, fmt).isoformat(timespec=timespec).split('+')[0] + UTC_Z

