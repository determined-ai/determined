import os
import re
from typing import Optional


def normalize_prefix(prefix: Optional[str]) -> str:
    new_prefix = ""
    if prefix is not None and prefix != "":
        banned_patterns = (r"^.*\/\.\.\/.*$", r"^\.\.\/.*", r".*\/\.\.$", r"^\.\.$")
        if any(re.match(bp, prefix) for bp in banned_patterns):
            raise ValueError(f"prefix must not match: {' '.join(banned_patterns)}")
        new_prefix = os.path.normpath(prefix).lstrip("/")
    return new_prefix
