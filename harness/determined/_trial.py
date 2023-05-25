import abc
from typing import Optional, Type

import determined as det


class LegacyTrial(metaclass=abc.ABCMeta):
    """
    Abstract base class for trials.

    A Trial is essentially a collection of user hooks that will be called by the TrialController.
    Frameworks should create framework-specific subclasses to specify framework-specific hooks.
    """

    # trial_controller_class specifies the subclass of TrialController that is
    # used in training for a given subclass of Trial.
    trial_controller_class = None  # type: Optional[Type['det.TrialController']]

    # trial_context_class specifies the subclass of TrialContext that is used in
    # training for a given subclass of Trial.
    trial_context_class = det.TrialContext  # type: Type[det.TrialContext]

    @abc.abstractmethod
    def __init__(self, context: det.TrialContext) -> None:
        """
        Initializes a trial using the provided ``context``.

        Override this method to initialize any common state that is shared
        by the other methods in the trial class. it is also typically useful
        to store ``context`` as an instance variable so that it can be accessed
        by other methods.
        """
        pass
