# link for backwards compatibility, since this was the original place that Session was exposed.
# At the present time, users should be using `determined.experimental.client.Session` instead, but
# since there's a big breaking change after we remove `client` from `determined.experimental`, we
# should remove this at that time.
# TODO: remove this link when we remove `client` from `determined.experimental`.

from determined.common.api import Session  # noqa: F401
