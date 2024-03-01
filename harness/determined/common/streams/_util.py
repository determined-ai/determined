from typing import Iterable


def range_encoded_keys(known: str) -> Iterable[int]:
    if not known:
        return
    for spec in known.split(","):
        if "-" in spec:
            # A range.
            start, end = spec.split("-")
            for i in range(int(start), int(end) + 1):
                yield i
        else:
            # A single deletion.
            yield int(spec)
