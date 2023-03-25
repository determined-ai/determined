# Determined docs development

This document lists some of the enhancements Determined has made to its
Sphinx-generated documentation.

## `make live`

When creating and editing files, you can use `make live` to simulate live docs
and automatically rebuild the docs whenever you save a change.

To try it out, run `make live` in a terminal and visit `http://localhost:1234`
in your browser.

If `make live` fails or is killed, the browser window will crash itself so that
you don't continue making edits and wonder why your edits aren't appearing in
your browser.

## Algolia search

[Aloglia](https://www.algolia.com) search is a search-as-a-service provider.
They index our site via their [crawler](https://crawler.algolia.com/admin),
then we include their search bar component into our website, so that searches
in the search bar obtain results based on Algolia's hosted search index for our
site.

We configure and inject the Algolia search bar into each page via
[this JavaScript](assets/scripts/docsearch.sbt.js).

### Relative search results

By default, Algolia search results return absolute URLs to
`https://docs.determined.ai`, based on the actual URL of searched pages.  The
effect is that somebody searching the docs hosted on their on-prem cluster
would be redirected to `https://docs.determined.ai` when they click on a search
result.  This would be very annoying for users.

Fortunately, Algolia allows us to define a `transformItems` function that can
make arbitrary client-side changes to the search results before displaying them
to a user.  The relativization happens in the JS code, but the key feature
required to relativize the search results is to know the relative path from the
current page to the root of the docs.  We embed this into every docs page's
header as a special `rel=root` link inside
[one of our template overrides](_templates/page.html).

### Versioning

You can not only view the latest docs, but you can also view older doc
versions.  The search results you see should be for the version of the docs you
are viewing.  Therefore, we configured the Algolia crawler to be version-aware.
The crawler sets the `version` tag on all results, which it extracts from the
URL path it is crawling.

Since Algolia doesn't allow filtering by arbitrary tags, we also added the
`version` tag to `determined` Algoila index's `attributesForFaceting` setting,
as a "filter-only" facet.

To actually filter by version during a search, we need to know which version of
the docs the current page was built against.  We embed this information into
the same `rel=root` link mentioned above.

### Sphinx-native search fallback

Determined have some users whose clusters do not have internet access.  For
those users Algolia's search-as-a-service model will never work.  When we
detect that we are unable to load resources from aloglia, we fall back to the
default Sphinx search.  See the JavaScript for implementation.

### Dev builds

Because Algolia indexes after docs are published to docs.determined.ai,
development builds of docs cannot rely on a versioned Algolia index.  Instead,
dev builds search the `latest` Algolia index, which is based on the most recent
_published_ version of Determined.

That means dev builds of docs will never return quite the right results. Builds
of unreleased docs will have out-of-date search results, while builds of old
docs will have search results from the future.  Even so, because the Sphinx
search results are so bad, this is a tradeoff we are willing to accept.

### Canonical URLs

Our site configures canonical links to point to `/latest` all the time.  This
is necessary for optimizing SEO, and is a common practice on other docs sites
(python standard library docs, for instance).  As a result, the Algolia crawler
must be configured with `ignoreCanonicalTo: true` before it will index anything
other than `/latest`.
