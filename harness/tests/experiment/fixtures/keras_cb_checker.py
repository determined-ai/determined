import io
from typing import Any, Callable, Dict, List, Optional, Set

import determined as det
import determined.keras

train_begin = "on_train_begin"
train_workload_begin = "on_train_workload_begin"
train_batch_begin = "on_train_batch_begin"
train_batch_end = "on_train_batch_end"
train_workload_end = "on_train_workload_end"
test_begin = "on_test_begin"
test_batch_begin = "on_test_batch_begin"
test_batch_end = "on_test_batch_end"
test_end = "on_test_end"
epoch_begin = "on_epoch_begin"
epoch_end = "on_epoch_end"
train_end = "on_train_end"
get_state = "get_state"
load_state = "load_state"

all_checks = []


def cb_check(fn: Callable) -> Callable:
    all_checks.append(fn)
    return fn


def do_check_with_table(lines: List[str], transitions: Dict[str, Set[Optional[str]]]) -> None:
    state = None
    i_prev = None
    for i, line in enumerate(lines):
        cb = line.split(":")[0]
        if cb in transitions:
            assert (
                state in transitions[cb]
            ), f"illegal callback {cb} on line {i} after {state} on line {i_prev}"
            state = cb
            i_prev = i


@cb_check
def check_train_begin_and_end(lines: List[str], **kwargs: Dict) -> None:
    assert lines[0].startswith(train_begin), "first call was not on_train_begin"
    assert lines[-1].startswith(train_end), "last call was not on_train_end"


@cb_check
def check_pause_continues(lines: List[str], **kwargs: Dict) -> None:
    do_check_with_table(
        lines,
        {
            train_begin: {None},
            load_state: {train_begin, get_state},
            get_state: {train_begin, load_state, get_state},
            train_end: {train_begin, load_state, get_state},
        },
    )


@cb_check
def check_initial_calls(lines: List[str], **kwargs: Dict) -> None:
    """
    Always expect the following commands before the first training
      - on_epoch_begin
      - on_train_workload_begin
      - on_validation_period_begin

    (except in the case of continued training, of course)
    """
    expect = {epoch_begin, train_workload_begin}
    remain = {epoch_begin, train_workload_begin}

    for i, line in enumerate(lines):
        cb = line.split(":")[0]
        if cb in expect:
            assert cb in remain, f"got two {cb} on line {i}"
            remain.remove(cb)
        elif line.startswith("on_train_batch_begin"):
            assert len(remain) == 0, f"still expecting {remain} on line {i}"
            break


@cb_check
def check_train_callbacks(lines: List[str], **kwargs: Dict) -> None:
    """
    All test_{begin,end} calls should be wrapped within validation_period_{begin,end}.

    (This assumes we only have validation_period-related model.evaluate() calls.)
    """
    do_check_with_table(
        lines,
        {
            train_begin: {None},
            train_workload_begin: {train_begin, train_workload_end},
            train_batch_begin: {train_workload_begin, train_batch_end},
            train_batch_end: {train_batch_begin},
            train_workload_end: {train_batch_end},
            train_end: {train_workload_end},
        },
    )


@cb_check
def check_test_callbacks(lines: List[str], **kwargs: Dict) -> None:
    """
    All test_{begin,end} calls should happen outside of train_workload_{begin,end} periods.
    """
    do_check_with_table(
        lines,
        {
            train_begin: {None},
            train_workload_begin: {train_begin, train_workload_end, test_end},
            train_workload_end: {train_workload_begin},
            test_begin: {train_begin, train_workload_end},
            test_batch_begin: {test_begin, test_batch_end},
            test_batch_end: {test_batch_begin},
            test_end: {test_batch_end},
            train_end: {test_end, train_workload_end},
        },
    )


@cb_check
def check_sane_epochs(lines: List[str], **kwargs: Dict) -> None:
    do_check_with_table(
        lines,
        {
            train_begin: {None},
            epoch_begin: {train_begin, epoch_end},
            epoch_end: {epoch_begin},
        },
    )


@cb_check
def check_validation_and_epoch_counts(lines: List[str], **kwargs: Dict) -> None:
    """Ensure that validation_period and epoch indices increment by 1 each time."""
    counts = {"on_epoch": 0, "on_validation_period": 0}
    for i, line in enumerate(lines):
        cb = line.split(":")[0]
        for prefix in counts:
            if cb.startswith(prefix):
                cb_idx = int(line.split(":")[1])
                if cb.endswith("begin"):
                    "got on_validation_period_begin for index 0 but expected 0 on line {i}"
                    assert (
                        cb_idx == counts[prefix]
                    ), f"got {cb} for index {cb_idx} but expected {counts[prefix]} on line {i}"
                if cb.endswith("end"):
                    assert (
                        cb_idx == counts[prefix]
                    ), f"got {cb} for index {cb_idx} but expected {counts[prefix]} on line {i}"
                    counts[prefix] += 1


@cb_check
def check_epoch_ends(lines: List[str], **kwargs: Dict) -> None:
    """Ensure the correct number of epochs were called"""
    count = kwargs.get("epochs")
    if not isinstance(count, int):
        return
    seen = 0
    for i, line in enumerate(lines):
        if line.startswith(epoch_end):
            seen += 1
            assert seen <= count, f"saw {epoch_end} {seen} on line {i} but expected only {count}"
    assert seen == count, f"expected {count} {epoch_end} calls but only saw {seen}"


@cb_check
def check_test_ends(lines: List[str], **kwargs: Dict) -> None:
    """Ensure the correct number of validation period ends were called"""
    count = kwargs.get("validations")
    if not isinstance(count, int):
        return
    seen = 0
    for i, line in enumerate(lines):
        if line.startswith(test_end):
            seen += 1
            assert seen <= count, f"saw {test_end} {seen} on line {i} but expected only {count}"
    assert seen == count, f"expected {count} {test_end} calls but only saw {seen}"


class CBChecker(det.keras.callbacks.Callback):
    def __init__(self, epochs: Optional[int] = None, validations: Optional[int] = None) -> None:
        super().__init__()
        self.log = io.StringIO()
        self.epochs = epochs
        self.validations = validations

    def on_train_begin(self, logs: Optional[Dict]) -> None:
        print(f"{train_begin}:{logs}", file=self.log)

    def on_train_end(self, logs: Optional[Dict]) -> None:
        print(f"{train_end}:{logs}", file=self.log)

        lines = self.log.getvalue().splitlines()

        try:
            for check in all_checks:
                check(lines, epochs=self.epochs, validations=self.validations)
        except AssertionError:
            for i, line in enumerate(lines):
                print(f"{i}:\t{line}")
            raise

    def on_test_begin(self, logs: Optional[Dict]) -> None:
        print(f"{test_begin}:{logs}", file=self.log)

    def on_test_end(self, logs: Optional[Dict]) -> None:
        print(f"{test_end}:{logs}", file=self.log)

    def on_epoch_begin(self, epoch: int, logs: Optional[Dict]) -> None:
        print(f"{epoch_begin}:{epoch}:{logs}", file=self.log)

    def on_epoch_end(self, epoch: int, logs: Optional[Dict]) -> None:
        print(f"{epoch_end}:{epoch}:{logs}", file=self.log)

    def on_train_batch_begin(self, batch: int, logs: Optional[Dict]) -> None:
        print(f"{train_batch_begin}:{batch}:{logs}", file=self.log)

    def on_train_batch_end(self, batch: int, logs: Optional[Dict]) -> None:
        print(f"{train_batch_end}:{batch}:{logs}", file=self.log)

    def on_test_batch_begin(self, batch: int, logs: Optional[Dict]) -> None:
        print(f"{test_batch_begin}:{batch}:{logs}", file=self.log)

    def on_test_batch_end(self, batch: int, logs: Optional[Dict]) -> None:
        print(f"{test_batch_end}:{batch}:{logs}", file=self.log)

    def on_train_workload_begin(
        self, batches_trained: int, batches_requested: Optional[int], logs: Dict
    ) -> None:
        print(f"{train_workload_begin}:{batches_trained}:{batches_requested}:{logs}", file=self.log)

    def on_train_workload_end(self, batches_trained: int, logs: Dict) -> None:
        print(f"{train_workload_end}:{batches_trained}:{logs}", file=self.log)

    def get_state(self) -> Any:
        print(f"{get_state}:", file=self.log)
        return self.log.getvalue()

    def load_state(self, state: Any) -> None:
        self.log = io.StringIO(state)
        # Seek to the end of the StringI0.
        pos, whence = 0, 2
        self.log.seek(pos, whence)
        print(f"{load_state}:", file=self.log)
