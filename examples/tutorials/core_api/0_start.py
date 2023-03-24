"""
Let's pretend our goal is to increment an integer over and over.  It'll be fun.

Stage 0 is intended to look like a completely unmodified "training" script.
Presumably you already have your own training script, that's a lot more complex
than this.  By following these examples you can get an idea of how to modify
your own script to integrate with the Core API.
"""

import logging
import sys
import time


def main(increment_by):
    x = 0
    for batch in range(100):
        x += increment_by
        time.sleep(0.1)
        print(f"x is now {x}")


if __name__ == "__main__":
    main(increment_by=1)
