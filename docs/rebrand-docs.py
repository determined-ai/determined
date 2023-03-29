#!/usr/bin/env python3

import glob
import os
import re
import sys
import time
from concurrent import futures

import docutils.nodes
from rstfmt import debug, rst_extras, rstfmt

EXCLUDE = {"requirements.txt"}


def sub_multi(repls, t):
    for a, b in repls:
        t = re.sub(a, b, t)
    return t


def rewrite_text(t):
    c = "HPE Machine Learning Development Environment"
    return sub_multi(
        [
            (r"Determined\s+AI", "HPE"),
            (r"A\s+Determined", f"An {c}"),
            (r"a\s+Determined", f"an {c}"),
            (r"Determined EE", c),
            (r"Determined", c),
            (r"\(EE-only\) ", ""),
        ],
        t,
    )


OrigFormatters = rstfmt.Formatters
OrigCodeFormatters = rstfmt.CodeFormatters


class CodeFormatters(OrigCodeFormatters):
    @staticmethod
    def python(code: str) -> str:
        code = "\n".join(
            rewrite_text(line) if line.lstrip().startswith("#") else line
            for line in code.splitlines()
        )

        return OrigCodeFormatters.python(code)


class Formatters(OrigFormatters):
    @staticmethod
    def title(node: docutils.nodes.title, ctx: rstfmt.FormatContext) -> rstfmt.line_iterator:
        text = " ".join(rstfmt.wrap_text(None, rstfmt.chain(rstfmt.fmt_children(node, ctx))))
        if text == "HPE":
            text = "HPE Machine Learning Development Environment"
        char = rstfmt.section_chars[ctx.section_depth - 1]
        if ctx.section_depth <= rstfmt.max_overline_depth:
            line = char * (len(text) + 2)
            yield line
            yield " " + text
            yield line
        else:
            yield text
            yield char * len(text)

    @staticmethod
    def Text(node: docutils.nodes.Text, _: rstfmt.FormatContext) -> rstfmt.inline_iterator:
        exclude_types = (docutils.nodes.literal_block, docutils.nodes.literal)
        if isinstance(node.parent, exclude_types) or isinstance(node.parent.parent, exclude_types):
            yield node.astext()
        # The rawsource attribute tends not to be set for text nodes not directly under paragraphs.
        elif isinstance(node.parent, docutils.nodes.paragraph):
            # Any instance of "\ " disappears in the parsing. It may have an effect if it separates
            # this text from adjacent inline markup, but in that case it will be replaced by the
            # wrapping algorithm. Other backslashes may be unnecessary (e.g., "a\` b" or "a\b"), but
            # finding all of those is future work.
            yield rewrite_text(node.rawsource.replace(r"\ ", ""))
        else:
            yield rewrite_text(node.astext())

    @staticmethod
    def ref_role(node: docutils.nodes.Node, ctx: rstfmt.FormatContext) -> rstfmt.inline_iterator:
        a = node.attributes
        target = a["target"]

        non_rewrite_roles = {"class", "func", "meth"}
        rewrite = (lambda x: x) if a["name"] in non_rewrite_roles else rewrite_text

        if a["has_explicit_title"]:
            title = a["title"].replace("<", r"\<")
            # TODO: This is a bit too broad, but not incorrect.
            title = rewrite(title.replace("`", r"\`"))
            text = f"{title} <{target}>"
        else:
            text = rewrite(target)
        yield rstfmt.inline_markup(f":{a['name']}:`{text}`")


rstfmt.Formatters = Formatters
rstfmt.CodeFormatters = CodeFormatters


def do_file(in_dir, out_dir, in_fn):
    rel = os.path.relpath(in_fn, in_dir)
    if rel in EXCLUDE:
        return
    out_fn = os.path.join(out_dir, rel)

    with open(in_fn) as f:
        text = f.read()
    doc = rstfmt.parse_string(text)
    os.makedirs(os.path.dirname(out_fn), exist_ok=True)
    with open(out_fn, "w") as f:
        print(rstfmt.format_node(100, doc), file=f, end="")


def edit_conf_py(in_dir, out_dir):
    try:
        with open(os.path.join(in_dir, "conf.py")) as f:
            text = f.read()
    except FileNotFoundError:
        return

    extra_text = time.strftime(
        """
project = "HPE Machine Learning Development Environment"
html_title = project + " Documentation"
copyright = "%Y, HPE"
author = "HPE Machine Learning Development Environment"

html_css_files = [
    "https://cdn.jsdelivr.net/npm/@docsearch/css@3",
    "styles/determined.css",
    "styles/hpe.css",
]
html_favicon = "assets/images/favicon-hpe.ico"
html_theme_options = {
    "logo": {
        "image_light": "assets/images/logo-hpe-on-light-horizontal.svg",
        "image_dark": "assets/images/logo-hpe-on-dark-horizontal.svg",
    },
    "repository_url": "https://github.com/determined-ai/determined",
    "use_repository_button": True,
    "use_download_button": False,
    "use_fullscreen_button": False,
}
html_baseurl = "https://hpe-mlde.determined.ai"
"""
    )

    with open(os.path.join(out_dir, "conf.py"), "w") as f:
        f.write(text + extra_text)


def main(args):
    rst_extras.register()
    in_dir = args[0]
    out_dir = args[1]

    jobs = []
    with futures.ProcessPoolExecutor() as pool:
        for in_fn in glob.glob(os.path.join(in_dir, "*.rst")) + glob.glob(
            os.path.join(in_dir, "**", "*.rst")
        ):
            jobs.append(pool.submit(do_file, in_dir, out_dir, in_fn))

    for job in jobs:
        job.result()

    edit_conf_py(in_dir, out_dir)


if __name__ == "__main__":
    exit(main(sys.argv[1:]))
