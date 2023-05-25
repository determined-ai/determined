#!/usr/bin/env python3

"""
redirects.py: a tool for preserving old links in new docs.

Usage:
    redirects.py mv src... dst     # like git mv, but also tracks redirects
    redirects.py check             # check redirects against filesystem for mistakes
    redirects.py redirect src dst  # manually configure a single redirect without moving files
"""

import argparse
import collections
import itertools
import json
import os
import pathlib
import subprocess
import sys

HERE = pathlib.Path(__file__).parent
DIR = HERE / ".redirects"
REDIRECTS = DIR / "redirects.json"
ALL_URLS = DIR / "all_published_urls_ever.json"
UNDO = DIR / "undo.json"

# colors
red = "\x1b[31m"
ylw = "\x1b[33m"
res = "\x1b[m"


def links_from_json(obj):
    """
    Example line from sphinx-formatted redirects:

        "training-apis/best-practices": "../index.html",

    Note that the key is the src and the val is the dst, relative to the src.

    Also note that the key has no extension but the value does.

    Instead, we would like to work with paths which are always relative to the docs/ dir,
    and we would like to not have to think about extensions.
    """
    out = []
    for key, val in obj.items():
        fulldst = os.path.normpath(os.path.join(os.path.dirname(key), val))
        dst, _ = os.path.splitext(fulldst)
        out.append(Link(key, dst))
    return out


class Link:
    def __init__(self, src, dst):
        self.src = src
        self.dst = dst

    def __repr__(self):
        return f"Link(src={repr(self.src)}, dst={repr(self.dst)})"

    def to_sphinx(self):
        """Inverse of links_from_json()."""
        dst = os.path.relpath(self.dst, os.path.dirname(self.src))
        return self.src, dst + ".html"

    def render(self):
        val = os.path.relpath(self.dst, os.path.dirname(self.src))
        return f'    "{self.src}": "{self.dst}.html",'


def doc_exists(name):
    return os.path.exists(name + ".md") or os.path.exists(name + ".rst")


def check_links(links):
    errors = []
    srcs = set()
    dsts = {}
    for l in links:
        srcs.add(l.src)
        dsts.setdefault(l.dst, []).append(l.dst)
        if not doc_exists(l.dst):
            errors.append(f"broken redirect detected: {l}")
        if doc_exists(l.src):
            errors.append(f"redirect shadows an actual file: {l}")

    for path in srcs.intersection(set(dsts)):
        errors.append(f"multi-layer redirect detected, {l} points to another redirection")

    return errors


def rename_one(links, published, fullsrc, fulldst):
    src, src_ext = os.path.splitext(fullsrc)
    dst, dst_ext = os.path.splitext(fulldst)

    out = []

    if src in published:
        # This src was published at some point, so emit a new redirect to preserve the url.
        l = Link(src, dst)
        print(ylw + f"adding new {l}" + res, file=sys.stderr)
        out.append(Link(src, dst))
    else:
        # This src was never published and we don't care about it.
        print(ylw + f"omitting new redirect for unpublished url {src}" + res, file=sys.stderr)

    # Update existing links that may be affected.
    for l in links:
        if l.src == src:
            # This should not happen; we have checks to prevent links shadowing real files.
            raise RuntimeError("detected link shadowing existing file in rename()")
        if l.src == dst:
            # The new filename shadows an old redirect, which we should just drop.
            print(ylw + f"dropping {l} which would shadow dst={dst}" + res, file=sys.stderr)
            continue
        if l.dst == src:
            # Link used to point to this file, now it should point to the new location.
            print(ylw + f"correcting {l} to point to dst={dst}" + res, file=sys.stderr)
            out.append(Link(l.src, dst))
            continue
        if l.dst == dst:
            # The old link is probably not pointing to the right thing anymore.
            print(red + f"Old {l} may not make sense anymore" + res, file=sys.stderr)
        # Leave link unmodified.
        out.append(l)
    return out


def rename_into(links, published, src, dst, drop_src_root=False):
    """
    Implement renaming many things into a directory.

    drop_src_root=True indicates a `mv thing newdir/`, where `thing` isn't going to appear in the
    final destination path for any elements of `thing/`, because it will be called `newdir/`
    isntead.
    """
    if not os.path.isdir(src):
        # src=basepath/base.rst, dst=dstpath/dst
        # result=dstpath/dst/base.rst
        result = os.path.join(dst, os.path.basename(src))
        return rename_one(links, published, src, result)
    for root, _, files in os.walk(src):
        for file in files:
            # src=basepath/base/, dst=dstpath/dst
            # root=basepath/base/a/b/c, file=d
            # fullsrc=basepath/base/a/b/c/d
            # fulldst=dstpath/dst/[base/]a/b/c/d
            fullsrc = os.path.relpath(os.path.join(root, file), start=HERE)
            if drop_src_root:
                relsrc = os.path.relpath(fullsrc, start=src)
                fulldst = os.path.join(dst, relsrc)
            else:
                fulldst = os.path.join(dst, fullsrc)
            links = rename_one(links, published, fullsrc, fulldst)
    return links


def write_json(obj, path):
    tmp = DIR / "tmp"
    with tmp.open("w") as f:
        out = dict(l.to_sphinx() for l in links)
        json.dump(obj, f, indent="    ")
        f.write("\n")
    tmp.rename(path)


def all_urls_from_files():
    urls = set()
    for root, dirs, files in os.walk(HERE):
        if os.path.relpath(root, HERE) in [
            "release-notes",
            "site",
            "build",
            "assets",
            "_templates",
        ]:
            while dirs:
                dirs.pop(0)
            continue
        for file in files:
            base, ext = os.path.splitext(file)
            if ext not in [".rst", ".md"]:
                continue
            if file == "README.md":
                continue
            url = os.path.relpath(os.path.join(root, base), HERE)
            urls.add(url)
    return urls


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(__doc__, file=sys.stderr)
        exit(1)

    rtext = REDIRECTS.read_text()
    links = links_from_json(json.loads(rtext))

    atext = ALL_URLS.read_text()
    published = set(json.loads(atext))

    if sys.argv[1] == "redirect":
        if len(sys.argv) != 4:
            print(__doc__, file=sys.stderr)
            exit(1)

        src = sys.argv[2]
        dst = sys.argv[3]

        print(f"redirecting {src} to {dst}", file=sys.stderr)

        links = rename_one(links, published, src, dst)
        write_json(dict(l.to_sphinx() for l in links), REDIRECTS)

        # Check links afterwards.
        errors = check_links(links)
        if errors:
            print("Warning: errors detected!", file=sys.stderr)
            for e in errors:
                print(e, file=sys.stderr)
            exit(1)

        exit(0)

    if sys.argv[1] == "mv":
        if not os.path.samefile(HERE, os.getcwd()):
            # Script is not tested or even expected to work from any other path.  Keep it simple.
            print(f"mv subcommand may only be called from {HERE}/", file=sys.stderr)
            exit(1)

        if len(sys.argv) < 4:
            print(__doc__, file=sys.stderr)
            exit(1)

        srcs = sys.argv[2:-1]
        dst = sys.argv[-1]

        for src in srcs:
            if not os.path.exists(src):
                print(f"source '{src}' does not exist", file=sys.stderr)
                exit(1)

        if len(srcs) > 1:
            # Multiple srcs means we must be moving files into a directory.
            if not os.path.isdir(dst):
                print(f"destination '{dst}' is not a directory", file=sys.stderr)
                exit(1)
            for src in srcs:
                links = list(rename_into(links, published, src, dst))
        elif not os.path.exists(dst):
            # dst does not exist, src will be renamed to become dst.
            if os.path.isdir(srcs[0]):
                # Rename a directory.
                links = rename_into(links, published, srcs[0], dst, drop_src_root=True)
            else:
                # Rename a file.
                links = rename_one(links, published, srcs[0], dst)
        else:
            # dst does exist, and must be a directory... we don't do file overwrites here.
            if not os.path.isdir(dst):
                print(f"destination '{dst}' already exists", file=sys.stderr)
                exit(1)
            links = rename_into(links, published, srcs[0], dst)

        # Affect the git repo to match our redirect change.
        cmd = ["git", "mv", *srcs, dst]
        print(f"{ylw}running `{' '.join(cmd)}`{res}", file=sys.stderr)
        subprocess.run(cmd, check=True)

        # Write output.
        write_json(dict(l.to_sphinx() for l in links), REDIRECTS)

        # Check links afterwards.
        errors = check_links(links)
        if errors:
            print("Warning: errors detected!", file=sys.stderr)
            for e in errors:
                print(e, file=sys.stderr)
            exit(1)

        exit(0)

    # Remaining commands require exactly one positional arg.
    if len(sys.argv) > 2:
        print(__doc__, file=sys.stderr)
        exit(1)

    if sys.argv[1] == "check":
        errors = check_links(links)
        if errors:
            print("check failed, errors detected!", file=sys.stderr)
            for e in errors:
                print(e, file=sys.stderr)
            exit(1)
        exit(0)

    if sys.argv[1] == "publish":
        """
        publish command is not in help text to keep the tool simple for docs writers.

        publish adds all current urls (both existing redirects and existing files) to the
        all_urls_published_ever.json file.

        publish shall be run as part of the release process.
        """
        # Not documented by help text because it's not meant to be used by docs writers.
        all_urls = set(l.src for l in links).union(all_urls_from_files())
        dropped = published.difference(all_urls)
        if dropped:
            print(
                "publish failed; the following previously-published urls seem to have been "
                "dropped:\n    " + "\n    ".join(dropped),
                file=sys.stderr,
            )
            exit(1)
        write_json(sorted(all_urls), ALL_URLS)
        exit(0)

    if sys.argv[1] == "publish-check":
        """
        publish-check command is not in help text to keep the tool simple for docs writers.

        publish-check checks that all current urls are present in all_urls_published_ever.json.

        publish-check shall be part of the CI for releases and release candidates.
        """

        all_urls = set(l.src for l in links).union(all_urls_from_files())
        if all_urls != published:
            print("publish-check failed", file=sys.stderr)
            dropped = published.difference(all_urls)
            extra = all_urls.difference(published)
            if dropped:
                print(
                    "The following previously-published urls seem to have been dropped:\n    "
                    + "\n    ".join(dropped),
                    file=sys.stderr,
                )
            if extra:
                print(
                    "The following newly-seen urls have not been marked as published:\n    "
                    + "\n    ".join(extra),
                    file=sys.stderr,
                )
            exit(1)
        exit(0)

    # No matching arg.
    print(__doc__, file=sys.stderr)
    exit(1)
