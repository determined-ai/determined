import enum

from determined.common.api import bindings


class OrderBy(enum.Enum):
    """
    Specifies whether a sorted list of objects should be in ascending or
    descending order.
    """

    ASCENDING = bindings.v1OrderBy.ASC.value
    ASC = bindings.v1OrderBy.ASC.value
    DESCENDING = bindings.v1OrderBy.DESC.value
    DESC = bindings.v1OrderBy.DESC.value

    def _to_bindings(self) -> bindings.v1OrderBy:
        return bindings.v1OrderBy(self.value)
