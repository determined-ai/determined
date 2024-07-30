#!/usr/bin/env python
import os
import sys

from daist.framework import venv

venv.make()
venv.activate()
os.execvp(sys.executable, [sys.executable, '-m', 'daist', '-k', 'daist'] + sys.argv[1:])
