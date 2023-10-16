from sphinx.addnodes import toctree as toctreenode

def sort_toctree_by_weight(app, doctree, docname):
    # Iterate through the toctree and sort based on weight
    for node in doctree.traverse(toctreenode):
        entries = node['entries']
        entries.sort(key=lambda entry: int(app.env.metadata[entry[1]].get('weight', 0)))


def setup(app):
    app.connect('doctree-resolved', sort_toctree_by_weight)
