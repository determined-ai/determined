# Documentation Guide

## Writing Documentation

Docs are generated using Sphinx. Documentation is written in reStructuredText, or in MyST Markdown.

The documentation build process is stored in the `Makefile`. Running `make live` will generate HTML formatted documentation.

Before submitting a pull request, run [rstfmt](https://pypi.org/project/rstfmt/).

## Using `make live`

When creating and editing files, you can use `make live` to simulate live docs
and automatically rebuild the docs whenever you save a change.

To try it out, run `make live` in a terminal and visit `http://localhost:1234`
in your browser.

If `make live` fails or is killed, the browser window will crash itself so that
you don't continue making edits and wonder why your edits aren't appearing in
your browser.

## Algolia Search

[Aloglia](https://www.algolia.com) search is a search-as-a-service provider.
They index our site via their [crawler](https://crawler.algolia.com/admin),
then we include their search bar component into our website, so that searches
in the search bar obtain results based on Algolia's hosted search index for our
site.

We configure and inject the Algolia search bar into each page via
[this JavaScript](assets/scripts/docsearch.sbt.js).

### Relative Search Results

By default, Algolia search results return absolute URLs to
`https://docs.determined.ai`, based on the actual URL of searched pages. The
effect is that somebody searching the docs hosted on their on-prem cluster
would be redirected to `https://docs.determined.ai` when they click on a search
result. This would be very annoying for users.

Fortunately, Algolia allows us to define a `transformItems` function that can
make arbitrary client-side changes to the search results before displaying them
to a user. The relativization happens in the JS code, but the key feature
required to relativize the search results is to know the relative path from the
current page to the root of the docs. We embed this into every docs page's
header as a special `rel=root` link inside
[one of our template overrides](_templates/page.html).

### Versioning

You can not only view the latest docs, but you can also view older doc
versions. The search results you see should be for the version of the docs you
are viewing. Therefore, we configured the Algolia crawler to be version-aware.
The crawler sets the `version` tag on all results, which it extracts from the
URL path it is crawling.

Since Algolia doesn't allow filtering by arbitrary tags, we also added the
`version` tag to `determined` Algoila index's `attributesForFaceting` setting,
as a "filter-only" facet.

To actually filter by version during a search, we need to know which version of
the docs the current page was built against. We embed this information into
the same `rel=root` link mentioned above.

### Sphinx-Native Search Fallback

Determined has some users whose clusters do not have internet access. For
those users, Algolia's search-as-a-service model will never work. When we
detect that we are unable to load resources from aloglia, we fall back to the
default Sphinx search. See the JavaScript for implementation.

### Dev Builds

Because Algolia indexes after docs are published to docs.determined.ai,
development builds of docs cannot rely on a versioned Algolia index. Instead,
dev builds search the `latest` Algolia index, which is based on the most recent
_published_ version of Determined.

That means dev builds of docs will never return quite the right results. Builds
of unreleased docs will have out-of-date search results, while builds of old
docs will have search results from the future. Even so, because the Sphinx
search results are so bad, this is a tradeoff we are willing to accept.

### Canonical URLs

Our site configures canonical links to point to `/latest` all the time. This
is necessary for optimizing SEO, and is a common practice on other docs sites
(python standard library docs, for instance). As a result, the Algolia crawler
must be configured with `ignoreCanonicalTo: true` before it will index anything
other than `/latest`.

## Theming

Our sphinx theme is a customized version of the `sphinx-book-theme`.

### Resources

- [sphinx-book-theme sample](https://sphinx-themes.org/sample-sites/sphinx-book-theme/)
- [sphinx-book-theme docs](https://sphinx-book-theme.readthedocs.io/en/latest/index.html)
- [PyData theme docs](https://pydata-sphinx-theme.readthedocs.io/en/latest/index.html): `sphinx-book-theme` uses many standards set in the PyData theme docs (for example, the names used for css variables)
- [Jinja docs](https://jinja.palletsprojects.com/en/3.0.x/templates/): templating language used by sphinx
- [Sphinx source](https://github.com/sphinx-doc/sphinx/tree/master/sphinx/themes/basic)

### Configuration

- `conf.py`
  - `html_sidebars` defines which template files should populate the sidebar on the left. `navbar-logo.html` and `sbt-sidebar-nav.html` come from `sphinx-book-theme` code. Additional template files (such as `sidebar-version.html` and `search-field.html`) are our custom additions, which live in the `_templates` folder.
  - `html_theme_options`: these are theme-specific options for configuring the logo, buttons, etc.
- Templates in `_templates` folder
  - `page.html` extends the default `page.html` that all sphinx sites have. You can overwrite a block using jinja syntax (`{% block block_name %}` `{% endblock %}`). The blocks available to overwrite can be seen in the Sphinx source ([sphinx/sphinx/themes/basic/layout.html](https://github.com/sphinx-doc/sphinx/blob/master/sphinx/themes/basic/layout.html), which `page.html` extends).
    - `analytics.html` in the `_templates` folder is included in the page via the `extrahead` block.
  - `search-field.html` and `sidebar-version.html` are our custom templates added to the sidebar, included in the page by listing them in `conf.py`'s `html_sidebars`.
  - `article-header-buttons.html` and `toggle-primary-sidebar.html` are overwriting template files specific to `sphinx-book-theme`. We customized these by copying the default content that exists in these parts of the page and then adding our own elements (not using block tags to extend).
    - The links in the header can be customized by editing the `header-links-right` element in `article-header-buttons.html` and the `header-links-left` element in `toggle-primary-sidebar.html`
- Styles in `assets/styles/determined.css`
  - `sphinx-book-theme` uses the same css variable names as PyData Theme. Partial list of available variables [here](https://pydata-sphinx-theme.readthedocs.io/en/latest/user_guide/styling.html).
