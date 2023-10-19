import re
import sys

from docutils import nodes
from sphinx.addnodes import toctree as toctreenode


def post_order_documents(app, env, docnames):
    # Order the documents to be processed so that children come before parents,
    # since the TOC-sorting code relies on the metadata for child documents
    # having already been read. (Just changing the priority of that hook doesn't
    # work, since that doesn't change that hooks for a given document are
    # executed all together, whereas we need to enforce ordering across
    # documents.)
    docnames.sort(key=lambda name: len(re.sub("_index$", "", name)), reverse=True)


def sort_toctree_by_weight(app, doctree, *args):
    def weight(doc):
        return int(app.env.metadata[doc].get("weight", 0))

    for node in doctree.traverse(toctreenode):
        node["entries"].sort(key=lambda entry: weight(entry[1]))
        node["includefiles"].sort(key=weight)


def setup(app):
    app.connect("env-before-read-docs", post_order_documents)
    
    # This needs to have a priority set so that it runs before
    # sphinx.environment.collectors.TocTreeCollector, which looks at the values
    # stored in the node and copies their order elsewhere for later usage.
    app.connect("doctree-read", sort_toctree_by_weight, priority=0)