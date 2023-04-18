/* Goal: convert the relative-to-docroot urls that we upload into the Algolia
   index into relative-to-the-current-page urls that a user can click.  Using
   relative urls means that even on-prem clusters can use Algolia search without
   being redirected to a different domain. */

// Find the path to the root index.html, from the special rel=root link.
// Example indexpath:  ../../index.html
const relroot = document.querySelectorAll("[rel=root]")[0];
// Extract the href as a string literal, to avoid url normalization.
const indexpath = relroot.attributes["href"].value;

// Our docroot is the directory containing index.html.
const docroot = indexpath.split("/").slice(0, -1).join("/");

// Extract the version, for picking the correct Algolia index.
let version = relroot.attributes["version"].value;
if (version.includes("-dev")) {
  /* Dev builds search against a special dev index that is update with every
     push to master. */
  version = "dev";
}else if (version.includes("-rc")) {
  /* Each release candidate publishes against the actual version without the
     "-rc" in the name. */
  version = version.slice(0, version.indexOf("-rc"));
}

try {
  docsearch({
    container: "#search-algolia",
    appId: "9H1PGK6NP7",
    indexName: "determined-" + version,
    apiKey: "18b6f7b0b2e20a6bdb00b660ff45d3b8",
    transformItems(items) {
      return items.map((item) => {
        // The url we scrape for the algolia index is relative to the docroot.
        item.url = docroot + "/" + item.url;
        return item;
      });
    },
  });
  // If the call to docsearch worked, hide the sphinx search bar.
  document.getElementById("search-fallback").style.display = "none";
} catch (e) {
  console.log(
    "falling back to sphinx search after configuring algolia failed:",
    e
  );
}
