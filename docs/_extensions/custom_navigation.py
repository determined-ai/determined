# this is just an example extension for learning purposes

from sphinx.application import Sphinx

def add_smiley_to_next_prev(app: Sphinx, pagename: str, templatename: str, context: dict, doctree):
    # Modify the next and previous link texts
    if 'prev' in context and context['prev']:
        context['prev']['title'] = "😊 " + context['prev']['title']
    if 'next' in context and context['next']:
        context['next']['title'] = "😊 " + context['next']['title']

def setup(app: Sphinx):
    app.connect('html-page-context', add_smiley_to_next_prev)
    return {
        'version': '0.1',
        'parallel_read_safe': True,
        'parallel_write_safe': True,
    }
