from sphinx.addnodes import toctree as toctreenode
import logging

# erase the log file each time
logging.basicConfig(filename='sphinx_log.txt', level=logging.DEBUG)
logger = logging.getLogger(__name__)

def sort_toctree_by_weight(app, doctree):
    # Iterate through the toctree and sort based on weight
    for node in doctree.traverse(toctreenode):

        # log the node 
        logger.debug(f"Node: {node}")
        # log the entries
        logger.debug(f"Node entries: {node['entries']}")

        entries = node['entries']
        logger.debug(f"Unsorted entries for {node}:")
        for entry in entries:

            doc_name = entry[1]
            all_metadata = app.env.metadata.get(doc_name, {})
            # log all the metadata for debugging
            logger.debug(f"     - Doc: {doc_name}, Metadata: {all_metadata}")
            # weight = all_metadata.get('weight', 0)
            # logger.debug(f"     - Doc: {doc_name}, Weight: {weight}, Description: {all_metadata.get('description', '')}")

        entries.sort(key=lambda entry: int(app.env.metadata[entry[1]].get('weight', 0)))
        # Log or print the sorted entries for debugging
        logger.debug(f"Sorted entries: {entries}")

def setup(app):
    app.connect('doctree-read', sort_toctree_by_weight)
