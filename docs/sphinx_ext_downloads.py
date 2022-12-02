"""
Sphinx is hardcoded to interpret links to downloadable files relative to the root of the docs
source tree. However, the downloadable files we want to use (tarballs of our examples directories)
are themselves generated at build time, and we would therefore like them to be separate from the
source. This module is a Sphinx plugin that replaces the normal interpretation of links, causing
Sphinx to look for downloads relative to a different directory (which is set in `conf.py`).
"""

import logging
import os
import types
from typing import Any, Dict

from docutils import nodes
from sphinx import addnodes, application
from sphinx.environment.collectors import asset
from sphinx.locale import __

logger = logging.getLogger(__name__)


class DownloadExternalFileCollector(asset.DownloadFileCollector):
    def process_doc(
        self: asset.DownloadFileCollector,
        app: application.Sphinx,
        doctree: nodes.document,
    ) -> None:
        """
        This function is different from the original method only in doing some surgery on the paths
        it finds when a separate root directory is configured.
        """
        for node in doctree.traverse(addnodes.download_reference):
            targetname = node["reftarget"]
            if "://" in targetname:
                node["refuri"] = targetname
            else:
                rel_filename, filename = app.env.relfn2path(targetname, app.env.docname)
                if app.config.dai_downloads_root:
                    filename = os.path.abspath(
                        os.path.join(app.config.dai_downloads_root, rel_filename)
                    )
                    rel_filename = os.path.relpath(filename, app.env.srcdir)
                app.env.dependencies[app.env.docname].add(rel_filename)
                if not os.access(filename, os.R_OK):
                    logger.warning(__("download file not readable: %s") % filename)
                    continue
                node["filename"] = app.env.dlfiles.add_file(app.env.docname, rel_filename)


def setup(app: application.Sphinx) -> Dict[str, Any]:
    app.add_config_value("dai_downloads_root", None, "html")

    # Disable the old instance of DownloadFileCollector and replace it with ours.
    for key in app.events.listeners:
        event = app.events.listeners[key]
        app.events.listeners[key] = [
            listener
            for listener in event
            if not (
                isinstance(listener.handler, types.MethodType)
                and isinstance(listener.handler.__self__, asset.DownloadFileCollector)
            )
        ]

    app.add_env_collector(DownloadExternalFileCollector)

    return {
        "version": "0",
        "parallel_read_safe": True,
        "parallel_write_safe": True,
    }
