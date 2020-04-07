import abc
from typing import Optional, Type

import determined as det


class Trial(metaclass=abc.ABCMeta):
    """
    Abstract base class for trials.

    A Trial is essentially a collection of user hooks that will be called by the TrialController.
    Frameworks should create framework-specific subclasses to specify framework-specific hooks.
    """

    # trial_controller_class specifies the subclass of TrialController that is
    # used in training for a given sublcass of Trial.
    trial_controller_class = None  # type: Optional[Type['det.TrialController']]

    # trial_context_class specifies the subclass of TrialContext that is used in
    # training for a given sublcass of Trial.
    trial_context_class = det.TrialContext  # type: Type[det.TrialContext]

    @abc.abstractmethod
    def __init__(self, trial_context: det.TrialContext) -> None:
        pass
