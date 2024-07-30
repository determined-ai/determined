import time
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import TestCase

from daist.models.result import FileMeta, Result
from daist.models.session import session
from daist.models.timestamp import UnixTime


class TestFileMeta(TestCase):
    def test_default_construction(self):
        file_meta = FileMeta()
        self.assertTrue(isinstance(file_meta.time, UnixTime))

    def test_time_setter(self):
        file_meta = FileMeta()
        time_sample = time.time()
        file_meta.time.unix = time_sample
        self.assertEqual(int(time_sample), file_meta.time.unix)


class TestResult(TestCase):
    def test_mv(self):
        result = Result()
        result.host.determined = session.determined.host
        with TemporaryDirectory() as tmp_src_dir, \
             TemporaryDirectory() as tmp_dst_dir:
            path_to_src_tmp = Path(tmp_src_dir)
            path_to_dst_tmp = Path(tmp_dst_dir)

            result.set_path(path_to_dst_tmp / result.get_filename())
            path_to_file = path_to_src_tmp / 'test_file'
            path_to_file.touch()
            result.mv(path_to_file, '.')
            self.assertFalse(path_to_file.exists())
            self.assertTrue((result.dir() / path_to_file.name).exists())

    def test_copyfile(self):
        result = Result()
        result.host.determined = session.determined.host

        with TemporaryDirectory() as src_tmp_dir, \
             TemporaryDirectory() as dst_tmp_dir:
            path_to_src_tmp = Path(src_tmp_dir)
            path_to_dst_tmp = Path(dst_tmp_dir)

            result.set_path(path_to_dst_tmp / result.get_filename())
            path_to_file = path_to_src_tmp / 'test_file'
            path_to_file.touch()

            self.assertFalse((result.dir() / path_to_file.name).exists())
            result.copyfile(path_to_file, '.')
            self.assertTrue(path_to_file.exists())
            self.assertTrue((result.dir() / path_to_file.name).exists())

