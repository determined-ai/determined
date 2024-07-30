from contextlib import contextmanager, redirect_stdout, redirect_stderr
from pathlib import Path
from tempfile import TemporaryDirectory
from typing import Tuple
from unittest import TestCase
import io
import logging

from daist.framework.paths import PkgPath
from daist.framework import log


class Test(TestCase):
    def test_levels(self):
        with _capture_output(logging.DEBUG, 'debug!') as output:
            stdout, stderr, file_data = output
            self.assertEqual('', stdout)
            self.assertEqual('', stderr)
            self.assertEqual('', file_data)

        with _capture_output(logging.INFO, 'info!') as output:
            stdout, stderr, file_data = output
            self.assertEqual('', stdout)
            self.assertEqual('', stderr)
            self.assertTrue('info!' in file_data)

        with _capture_output(logging.WARNING, 'warning!') as output:
            stdout, stderr, file_data = output
            self.assertTrue('warning!' in stdout)
            self.assertEqual('', stderr)
            self.assertTrue('warning!' in file_data)

        with _capture_output(logging.ERROR, 'error!') as output:
            stdout, stderr, file_data = output
            self.assertEqual('', stdout)
            self.assertTrue('error!' in stderr)
            self.assertTrue('error!' in file_data)

        with _capture_output(logging.CRITICAL, 'critical!') as output:
            stdout, stderr, file_data = output
            self.assertEqual('', stdout)
            self.assertTrue('critical!' in stderr)
            self.assertTrue('critical!' in file_data)

        try:
            1/0
        except ZeroDivisionError:
            with _capture_output(-1, 'exception!') as output:
                stdout, stderr, file_data = output
        self.assertEqual('', stdout)
        self.assertTrue('exception!' in stderr)
        self.assertTrue('traceback' in stderr.lower())
        self.assertEqual(stderr, file_data)


@contextmanager
def _capture_output(level: int, message: str) -> Tuple[str, str, str]:
    with (TemporaryDirectory() as tmp_dir,
          redirect_stdout(io.StringIO()) as stdout,
          redirect_stderr(io.StringIO()) as stderr):
        tmp_dir = Path(tmp_dir)
        path = tmp_dir / log.FILENAME
        log.start(path, logging.INFO, logging.WARNING)
        logger = logging.getLogger(PkgPath.PATH.name)

        if level < 0:
            logger.exception(message)
        else:
            logger.log(level, message)
        for hdlr in logger.handlers:
            hdlr.flush()

        stdout.seek(0)
        stderr.seek(0)
        with open(path, 'r') as inf:
            file_data = inf.read().strip()
        yield stdout.read().strip(), stderr.read().strip(), file_data
