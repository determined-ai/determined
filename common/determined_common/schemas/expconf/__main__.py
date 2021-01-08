import json
import sys

from determined_common.schemas import expconf

if __name__ == "__main__":
    example = json.load(sys.stdin)

    errors = expconf.validation_errors(example)

    if not errors:
        sys.exit(0)

    print("\n".join(expconf.validation_errors(example)))
    sys.exit(1)
