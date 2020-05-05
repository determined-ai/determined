from typing import Any, Callable

from determined import monkey_patch


def test_monkey_patch_context() -> None:
    class DummyModule:
        def return_one(self) -> int:
            return 1

    def return_two(orig_func: Callable, *args: Any, **kwargs: Any) -> int:
        return 2

    dummy_module = DummyModule()
    assert dummy_module.return_one() == 1
    with monkey_patch.monkey_patch(dummy_module, "return_one", return_two):
        assert dummy_module.return_one() == 2
        assert dummy_module.return_one.__name__ == DummyModule.return_one.__name__
    assert dummy_module.return_one() == 1
