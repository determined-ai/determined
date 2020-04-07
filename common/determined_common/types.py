import os
import sys
from typing import TYPE_CHECKING, NewType

# Make custom types show up as their underlying types in Sphinx docs --
# otherwise, using NewType puts a randomized address in their name that varies
# across runs, breaking Docker caching. Checking TYPE_CHECKING is necessary to
# prevent mypy from complaining about mismatched types between branches.
if not TYPE_CHECKING and os.path.basename(sys.argv[0]) == "sphinx-build":

    def NewType(name, tp):  # noqa: F811
        return tp


ExperimentID = NewType("ExperimentID", int)
TrialID = NewType("TrialID", int)
StepID = NewType("StepID", int)
