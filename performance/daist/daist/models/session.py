from typing import Optional, Tuple
from urllib.parse import urlparse
import logging
import subprocess
import shlex

from .config import Config
from .environment import environment
from .result import Result, FileMeta
from .timestamp import UnixTimeWithStamp
from ..framework import log
from ..framework.paths import PkgPath, VenvPath
from ..framework.typelib import Url_t
import daist


class Session:
    def __init__(self):
        self._cfg = Config.open(environment.daist_config)
        self._result = None

        det_host = environment.det_master
        if det_host is None:
            det_host = self._cfg.determined.det_master
        det_pass = environment.secrets.det_pass
        if det_pass is None:
            det_pass = self._cfg.determined.det_pass
        det_user = environment.det_user
        if det_user is None:
            det_user = self._cfg.determined.det_user
        self._determined = Determined(det_host, det_user, det_pass)

    @property
    def cfg(self) -> Config:
        return self._cfg

    @property
    def determined(self) -> 'Determined':
        return self._determined

    @property
    def result(self) -> Optional[Result]:
        return self._result

    def start(self):
        det_hostname = urlparse(self.determined.host).hostname

        self._result = Result()
        self._result.version.daist = daist.__version__
        self._result.version.determined.client, self._result.version.determined.server = \
            self._get_versions()
        self._result.host.determined = self._determined.host

        path_to_output_dir = self._cfg.exec.output
        if path_to_output_dir is None:
            path_to_output_dir = PkgPath.DEFAULT_RESULTS
        path_to_save_to = (path_to_output_dir /
                           det_hostname /
                           self._result.start_time.path_stamp /
                           self._result.get_filename())

        path_to_save_to.parent.mkdir(parents=True, exist_ok=True)
        self._result.set_path(path_to_save_to)

        file_level = getattr(logging, self.cfg.log.file_level, logging.INFO)
        stdout_level = getattr(logging, self.cfg.log.stdout_level, logging.WARNING)
        path_to_log = self._result.touch(log.FILENAME)
        log.start(path_to_log, file_level, stdout_level)

        file_meta = FileMeta()
        file_meta.time.unix = self._result.start_time.unix
        file_meta.test_id = None
        self._result.copyfile(self._cfg.path(), self._cfg.path().name, file_meta)

        self._result.save()

    def stop(self):
        self._result.end_time = UnixTimeWithStamp()
        self._result.save()

    @staticmethod
    def _get_versions() -> Tuple[str, str]:
        """
        .. todo:: Is there a way to get this information from the python API?
        :return:
        """
        proc = subprocess.run(shlex.split(f'{VenvPath.DET} -m'
                                          f' {session.determined.host} '
                                          f'version'),
                              stdout=subprocess.PIPE,
                              encoding='utf-8')
        client_version = ''
        server_version = ''
        in_client = False
        for line in proc.stdout.splitlines():
            line = line.strip()
            if line.startswith('client:'):
                in_client = True
            if line.startswith('master:'):
                in_client = False

            if line.startswith('version:'):
                if in_client:
                    client_version = line.split(':')[-1].strip()
                else:
                    server_version = line.split(':')[-1].strip()
        return client_version, server_version


class Determined:
    def __init__(self, host: Url_t, user: str, password: str):
        self._host = host
        self._user = user
        self._password = password

    @property
    def host(self) -> Url_t:
        return self._host

    @property
    def user(self) -> str:
        return self._user

    @property
    def password(self) -> str:
        return self._password


#: Application global singleton
session = Session()
