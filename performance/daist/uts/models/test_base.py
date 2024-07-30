from pathlib import Path
from tempfile import TemporaryDirectory
from unittest import TestCase

from daist.models.base import BaseObj, Format


class TestBase(TestCase):
    def test_save_and_open(self):
        class TestObj(BaseObj):
            def __init__(self, *_, **__):
                super().__init__()

        with self.assertRaises(Format.InvalidFormat):
            TestObj.open('some invalid path')

        with self.assertRaises(FileNotFoundError):
            TestObj().save()

        with TemporaryDirectory() as tmp_dir:
            path = Path(tmp_dir) / 'test.json'
            obj = TestObj()
            obj.save(path)
            obj.save()
            self.assertEqual(obj.path(), path)
            obj2 = TestObj.open(path)
            obj2.save()
