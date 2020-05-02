# Configuration file for the Sphinx documentation builder.
#
# This file only contains a selection of the most common options. For a full
# list see the documentation:
# http://www.sphinx-doc.org/en/master/config

# -- Path setup --------------------------------------------------------------

# If extensions (or modules to document with autodoc) are in another directory,
# add these directories to sys.path here. If the directory is relative to the
# documentation root, use os.path.abspath to make it absolute, like shown here.

import os
import pathlib
import sys

import determined_ai_sphinx_theme

sys.path.append(os.path.abspath(os.path.dirname(__file__)))

# -- Project information -----------------------------------------------------

project = "Determined"
html_title = "Determined AI Documentation"
copyright = "2020, Determined AI"
author = "hello@determined.ai"

# The version info for the project you"re documenting, acts as replacement for
# |version| and |release|, also used in various other places throughout the
# built documents.
#
# The short X.Y version.
version = pathlib.Path(__file__).parents[1].joinpath("VERSION").read_text()

# The full version, including alpha/beta/rc tags.
release = version

# -- General configuration ---------------------------------------------------

# Add any Sphinx extension module names here, as strings. They can be
# extensions coming with Sphinx (named 'sphinx.ext.*') or your custom
# ones.
extensions = [
    "sphinx_ext_downloads",
    "sphinx.ext.autodoc",
    "sphinx.ext.extlinks",
    "sphinx.ext.intersphinx",
    "sphinx.ext.mathjax",
    "sphinx.ext.napoleon",
    "sphinxarg.ext",
    "sphinx_gallery.gen_gallery",
    "sphinx_copybutton",
]

autosummary_generate = True
autoclass_content = "class"

# List of patterns, relative to source directory, that match files and
# directories to ignore when looking for source files.
# This pattern also effect to html_static_path and html_extra_path
exclude_patterns = ["_build", "Thumbs.db", ".DS_Store", "examples"]

# The suffix of source filenames.
source_suffix = {".rst": "restructuredtext", ".txt": "restructuredtext"}

# -- Options for HTML output -------------------------------------------------

# The theme to use for HTML and HTML Help pages.  See the documentation for
# a list of builtin themes.
#

# Add any paths that contain custom static files (such as style sheets) here,
# relative to this directory. They are copied after the builtin static files,
# so a file named 'default.css' will overwrite the builtin 'default.css'.
# html_static_path = ["_static"]

# Our custom sphinx extension uses this value to decide where to look for
# downloadable files.
builddir = os.environ.get("BUILDDIR", "../build")
dai_downloads_root = os.path.join(builddir, "docs-downloads")

# -- HTML theme settings ------------------------------------------------

html_show_sourcelink = False
html_show_sphinx = False
html_last_updated_fmt = None
html_sidebars = {"**": ["logo-text.html", "globaltoc.html", "localtoc.html", "searchbox.html"]}

html_theme_path = [determined_ai_sphinx_theme.get_html_theme_path()]
html_theme = "determined_ai_sphinx_theme"
html_logo = "assets/images/logo.png"
html_favicon = "assets/images/favicon.ico"

html_theme_options = {
    "analytics_id": "UA-110089850-1",
    "collapse_navigation": False,
    "display_version": True,
    "logo_only": False,
}

language = "en"

todo_include_todos = True

html_use_index = True
html_domain_indices = True

# -- Sphinx Gallery settings -------------------------------------------


class ExplicitOrder:
    """
    sphinx_gallery.sorting.ExplicitOrder doesn't work with
    within_subsection_order. Define a custom class for ordering examples within
    subsections with a static ordering.
    """

    ORDERING = {
        "native-tf-keras": [
            "tf_keras_native.py",
            "tf_keras_native_hparam_search.py",
            "tf_keras_native_dtrain.py",
        ]
    }

    def __init__(self, src_dir):
        self.gallery = pathlib.Path(src_dir).name
        if self.gallery not in ExplicitOrder.ORDERING:
            raise Exception("Ordering for gallery {} not found".format(self.gallery))

        self.ordering = ExplicitOrder.ORDERING[pathlib.Path(src_dir).name]

    def __call__(self, item):
        if item in self.ordering:
            return self.ordering.index(item)
        else:
            raise Exception(
                "Item '{}' not found in ordering for gallery {}".format(item, self.gallery)
            )


sphinx_gallery_conf = {
    "examples_dirs": "../examples/tutorials/native-tf-keras",
    "gallery_dirs": "tutorials/native-tf-keras",
    # Subsections are sorted by number of code lines per example. Override this
    # to sort via the explicit ordering.
    # "within_subsection_order": CustomOrdering,
    "within_subsection_order": ExplicitOrder,
    "download_all_examples": True,
    "plot_gallery": False,
    "min_reported_time": float("inf"),
}
