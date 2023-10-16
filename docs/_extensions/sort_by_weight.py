from sphinx.addnodes import toctree as toctreenode
import logging

logging.basicConfig(filename='sphinx_log.txt', level=logging.DEBUG)
logger = logging.getLogger(__name__)

def check_metadata(app, doctree):
    # Get the docname from the doctree's source attribute
    docname = app.env.path2doc(doctree.attributes['source'])
    
    # Get the metadata for the current document
    metadata = app.env.metadata.get(docname, {})
    
    # Log the metadata for debugging purposes
    logger.debug(f"Metadata for {docname}: {metadata}")
    
    # If you need to preprocess or modify the metadata, you can do it here.
    # For example, if you want to ensure that every document has a 'weight' metadata:
    if 'weight' not in metadata:
        metadata['weight'] = 0  # default value
        app.env.metadata[docname] = metadata


def sort_toctree_by_weight(app, doctree, docname):
    # Iterate through the toctree and sort based on weight
    for node in doctree.traverse(toctreenode):
        logger.debug(f"Node: {node}")
        entries = node['entries']
        for entry in entries:
            doc_name = entry[1]
            all_metadata = app.env.metadata.get(doc_name, {})
            logger.debug(f"     - Doc: {doc_name}, Metadata: {all_metadata}")

        entries.sort(key=lambda entry: int(app.env.metadata[entry[1]].get('weight', 0)))

def setup(app):
    app.connect('doctree-read', check_metadata)
    app.connect('doctree-resolved', sort_toctree_by_weight)
