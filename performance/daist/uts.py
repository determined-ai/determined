#!/usr/bin/env python
import os
import sys

from daist.framework import venv

venv.make()
venv.activate()
os.execvp(sys.executable, [sys.executable, '-Wignore', '-m', 'unittest', '-k', 'uts']
          + sys.argv[1:])
