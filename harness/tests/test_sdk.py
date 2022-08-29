from typing import Any, Optional

from determined.common.api import bindings
from determined.experimental import client


def do_enum_test(sdk_enum: Any, bindings_enum: Any, *, ignore: Optional[list] = None) -> None:
    # Every sdk enum member exists in bindings.
    extra_sdk_names = []
    for k, v in sdk_enum.__members__.items():
        if ignore and v in ignore:
            continue
        try:
            bindings_enum(v.value)
        except ValueError:
            extra_sdk_names.append(k)

    # Every bindings enum member exists in sdk.
    extra_bindings_names = []
    for k, v in bindings_enum.__members__.items():
        if ignore and v in ignore:
            continue
        try:
            sdk_enum(v.value)
        except ValueError:
            extra_bindings_names.append(k)

    errs = []
    if extra_sdk_names:
        errs.append(f"detected {extra_sdk_names} which are not valid bindings values\n")

    if extra_bindings_names:
        errs.append(f"detected {extra_bindings_names} which are not valid sdk values\n")

    assert not errs, " and ".join(errs)


def test_experiment_state() -> None:
    do_enum_test(client.ExperimentState, bindings.determinedexperimentv1State)


def test_trial_sort_by() -> None:
    do_enum_test(client.TrialSortBy, bindings.v1GetExperimentTrialsRequestSortBy)


def test_trial_order_by() -> None:
    do_enum_test(
        client.TrialOrderBy,
        bindings.v1OrderBy,
        # We don't give the user the UNSPECIFIED option.
        ignore=[bindings.v1OrderBy.ORDER_BY_UNSPECIFIED],
    )


def test_checkpoint_state() -> None:
    do_enum_test(client.CheckpointState, bindings.determinedcheckpointv1State)


def test_model_sort_by() -> None:
    do_enum_test(client.ModelSortBy, bindings.v1GetModelsRequestSortBy)


def test_model_order_by() -> None:
    do_enum_test(
        client.ModelOrderBy,
        bindings.v1OrderBy,
        # We don't give the user the UNSPECIFIED option.
        ignore=[bindings.v1OrderBy.ORDER_BY_UNSPECIFIED],
    )
