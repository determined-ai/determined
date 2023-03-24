/* Problem:

    We want to use Algolia-hosted search indices to search docs which might be
    hosted by on-prem clusters.

   Solution:

    We can use docsearch(transformItems=) to convert the absolute urls that
    Algolia sees into relative urls for navigating around docs wherever they
    may be hosted.

   Detail:

    To convert an absolute url to a relative url, we need to know:

     1) The relative doctree location of the result Algolia is returning.  This
        is trivial to calculate because we just have to subtract the '/VERSION/'
        bit of the result path from the result url.

     2) The path to the root of the docs, relative to this page.  This is
        calculated by a pathto('index') directive in one of our templates and
        embedded in the rel="root" we embed.
*/

// Find the path to the root index.html, from the special rel=root link.
// Example indexpath:  ../../index.html
const relroot = document.querySelectorAll("[rel=root]")[0]
// Extract the href as a string literal, to avoid url normalization.
const indexpath = relroot.attributes["href"].value;

// Our docroot is the directory containing index.html.
const docroot = indexpath.split("/").slice(0, -1).join("/");

// Extract the version, for filtering Algolia results.
let version = relroot.attributes["version"].value;
if(version.includes("-")){
    /* Dev builds search against the "latest" index, since there's not a
       great alternative. */
    version = "latest";
}
const searchParameters = {filters: 'version:"' + version + '"'};

try {
    docsearch({
        container: '#searchbox',
        appId: '9H1PGK6NP7',
        indexName: 'determined',
        apiKey: '18b6f7b0b2e20a6bdb00b660ff45d3b8',
        transformItems(items) {
            return items.map((item) => {
                const itempath = new URL(item.url).pathname;
                // Get the relative path based on what Algolia indexed.
                // Example: /latest/path/to/doc -> path/to/doc
                const itemrel = itempath.split("/").slice(2).join("/");
                // Point to the locally hosted version of the same document.
                if(docroot === ""){
                    item.url = itemrel
                } else {
                    item.url = docroot + "/" + itemrel;
                }
                return item;
            })
        },
        searchParameters: searchParameters,
    });
    // If the call to docsearch worked, hide the sphinx search bar.
    document.querySelectorAll(
        "[class=sidebar-search-container]"
    )[0].style.display = 'none';
} catch(e) {
    console.log(
        "falling back to sphinx search after configuring algolia failed:", e
    );
}
