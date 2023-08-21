import argparse
import html
import io
import json
import os
import pathlib
import re
import sys
import traceback
from xml.etree import ElementTree

from algoliasearch import search_client

HERE = pathlib.Path(__file__).parent

EXCLUDES = ["release-notes/", "attributions.xml"]

BUILD = str(HERE / ".." / "site" / "xml")

SETTINGS = {
    "minWordSizefor1Typo": 3,
    "minWordSizefor2Typos": 7,
    "hitsPerPage": 20,
    "minProximity": 1,
    "searchableAttributes": [
        "unordered(hierarchy.lvl0)",
        "unordered(hierarchy.lvl1)",
        "unordered(hierarchy.lvl2)",
        "unordered(hierarchy.lvl3)",
        "unordered(hierarchy.lvl4)",
        "unordered(hierarchy.lvl5)",
        "unordered(hierarchy.lvl6)",
        "content",
    ],
    "attributesToRetrieve": ["content", "hierarchy", "type", "url"],
    "allowTyposOnNumericTokens": False,
    "ignorePlurals": True,
    "advancedSyntax": True,
    "attributeCriteriaComputedByMinProximity": True,
    "distinct": True,
    "attributesToSnippet": ["content:10"],
    "attributesToHighlight": ["hierarchy", "content"],
    "paginationLimitedTo": 1000,
    "attributeForDistinct": "url",
    "exactOnSingleWordQuery": "attribute",
    "ranking": ["words", "filters", "typo", "attribute", "proximity", "exact", "custom"],
    "customRanking": ["desc(weight.pageRank)", "desc(weight.level)", "asc(weight.position)"],
    "separatorsToIndex": "",
    "removeWordsIfNoResults": "allOptional",
    "queryType": "prefixLast",
    "highlightPreTag": '<span class="algolia-docsearch-suggestion--highlight">',
    "highlightPostTag": "</span>",
    "alternativesAsExact": ["ignorePlurals", "singleWordSynonym"],
}


def mkrecord(hier, words, path, typ, order):
    hierarchy = {
        "lvl0": None,
        "lvl1": None,
        "lvl2": None,
        "lvl3": None,
        "lvl4": None,
        "lvl5": None,
        "lvl6": None,
    }
    for i, h in enumerate(hier):
        hierarchy[f"lvl{i}"] = h
    content = " ".join(html.unescape(w) for w in words) if words else None
    record = {
        "recordVersion": "v3",
        "hierarchy": hierarchy,
        # Upload a partial path relative to the docroot.  The docs javascript will combine the
        # path to the docroot with this partial path for a relative path from anywhere in the docs
        # to this result.
        "url": path,
        "content": content,
        "type": typ,
        "lang": "en",
        "language": "en",
        "weight": {
            "pageRank": 0,
            "level": 0 if content else 110 - 10 * len(hier),
            "position": order,
        },
    }
    return record


class ExtractionError(Exception):
    pass


def xmldumps(node):
    return ElementTree.tostring(node).decode("utf8")


def xml2str(node):
    return ElementTree.tostring(node, method="text").decode("utf8").strip()


def extract_file(root, xmlpath):

    # Workaround an incompatibility between sphinx's XML generation and ElementTree's XML parsing.
    #
    # The issue is references contain xml tags with colons in them:
    #
    #    <literal_strong py:class="True" py:module="determined.experimental.client" ...>
    #
    # Is that legal XML?  Who knows, who cares.  We don't need that information so we do the
    # simplest thing to make our ElementTree not puke, and that's remove those colons.
    #
    # But ElementTree doesn't have a parsestring() method, so we wrap the fixed text in BytesIO.
    with open(os.path.join(root, xmlpath), "rb") as f:
        raw = f.read()
        fixed = re.sub(b"py:([a-zA-Z_]*)=", b"py\\1=", raw)
        bytes_io = io.BytesIO(fixed)

    tree = ElementTree.parse(bytes_io)
    root = tree.getroot()
    # Extract page title.
    title_node = root.find("section").find("title")
    title = xml2str(title_node)
    assert title, xmldumps(title_node)

    htmlpath = xmlpath.replace(".xml", ".html")

    order = 1

    def _extract_section(node, hier_in, idx):
        nonlocal order
        name = None
        try:
            records = []
            words = []
            # Extract the anchor, the first of the "ids" attribute.
            anchor = node.attrib["ids"].split()[0]
            children = iter(node)
            # The first node must be the <title>.
            title_node = next(children)
            assert title_node.tag == "title", f"first node's tag was {title_node.tag}, not title"
            name = xml2str(title_node)
            assert name, xmldumps(title_node)
            hier = (*hier_in, name)
            # Find words for this section (whatever comes before the child sections).
            for child in children:
                if child.tag == "section":
                    have_subsections = True
                    break
                # Smush all the text together like we just don't care.
                words += xml2str(child).split()
            else:
                have_subsections = False

            # Make a header record with no content.
            path = f"{htmlpath}#{anchor}"
            records.append(mkrecord(hier, None, path, f"lvl{len(hier)-1}", order))
            order += 1
            if words:
                # Make a content-type record
                records.append(mkrecord(hier, words, path, "content", order))
                order += 1

            if have_subsections:
                records += _extract_section(child, hier, 0)
                for i, subsection in enumerate(children):
                    records += _extract_section(subsection, hier, i + 1)

            return records

        except Exception as e:
            if isinstance(e, ExtractionError):
                raise
            raise ExtractionError(
                f"extraction failed, hier_in={hier_in}, idx={idx}, name={name}, "
                f"node={xmldumps(node)[:512]}"
            ) from e

    return _extract_section(root.find("section"), [title], 0)


def scrape_tree(root, excludes):
    root = os.path.normpath(root)
    excludes = [os.path.normpath(os.path.join(root, ex)) for ex in excludes]
    records = []
    errors = []

    def exclude_dir(path):
        return any(os.path.relpath(path, ex) == "." for ex in excludes)

    def exclude_file(path):
        return any(path == ex for ex in excludes)

    for parent, dirs, files in os.walk(root):
        if exclude_dir(parent):
            print("skipping dir", parent, file=sys.stderr)
            dirs = []
            continue
        for file in files:
            _, ext = os.path.splitext(file)
            if ext != ".xml":
                continue
            path = os.path.join(parent, file)
            if exclude_file(path):
                print("skipping file", path, file=sys.stderr)
                continue
            try:
                records += extract_file(root, os.path.relpath(path, root))
            except Exception as e:
                errors.append(f"{path}: {traceback.format_exc()}")

    if errors:
        print("\n".join(errors), file=sys.stderr)
        exit(1)

    print(f"collected {len(records)} records", file=sys.stderr)

    return records


def upload(app_id, api_key, records, version):
    client = search_client.SearchClient.create(app_id, api_key)

    temp_name = f"determined-{version}.tmp"
    final_name = f"determined-{version}"

    # Create a temp index
    index = client.init_index(temp_name)

    # Pick some settings for this index.
    index.set_settings(SETTINGS)

    # Upload to temp index.
    print(f"uploading {len(records)} records to temp index {temp_name}...", file=sys.stderr)
    index.save_objects(records, {"autoGenerateObjectIDIfNotExist": True})
    print("upload complete", file=sys.stderr)

    # Rename index into place.
    print(f"renaming temp index {temp_name} -> {final_name}...", file=sys.stderr)
    client.move_index(temp_name, final_name).wait()
    print("rename done", file=sys.stderr)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--json", action="store_true", help="dump records to stdout")
    parser.add_argument("--upload", action="store_true", help="upload to algolia")
    parser.add_argument("--app-id", default="9H1PGK6NP7", help="algloia app id")
    parser.add_argument(
        "--api-key", default=os.environ.get("ALGOLIA_API_KEY"), help="algloia admin key"
    )
    args = parser.parse_args()

    # Pick the correct version.
    HERE = pathlib.Path(__file__).parent
    with (HERE / ".." / ".." / "VERSION").open() as f:
        version = f.read().strip()
    if "-dev" in version:
        # Dev builds search against a special dev index that is update with every push to master.
        version = "dev"
    elif "-rc" in version:
        # Each release candidate publishes against the actual version without the "-rc" in the name.
        version = version[: version.index("-rc")]

    records = scrape_tree(BUILD, EXCLUDES)

    if args.json:
        json.dump(records, sys.stdout, indent="  ")
        print()
        print(f"{len(records)} records for version={version} dumped to stdout")

    if args.upload:
        if not args.api_key:
            print("--api-key or ALGOLIA_API_KEY required for upload", file=sys.stderr)
            exit(1)
        upload(args.app_id, args.api_key, records, version)
        print(f"{len(records)} records uploaded for version={version}")

    if not args.upload and not args.json:
        print(f"scrape of version={version} succeeded, try --json or --upload", file=sys.stderr)
