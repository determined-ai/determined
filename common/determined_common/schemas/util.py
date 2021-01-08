from typing import Any, List


def _path_string(json_path: str) -> str:
    return "".join([f"[{p}]" if isinstance(p, int) else f".{p}" for p in json_path])


def _fmt_msg(e: Any) -> str:
    path = _path_string(e.absolute_path)
    return f"<config>{path}: {e.message}"


def format_validation_errors(errors: List) -> List[str]:
    return sorted(_fmt_msg(e) for e in errors)
