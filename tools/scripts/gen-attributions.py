#!/usr/bin/python3

"""
gen-attributions.py: a tool for generating various OSS attributions docs.

See tools/scripts/README.md for instructions on how to format license files.

usage:
    gen-attributions.py BUILD_TYPE > OUTPUT_FILE

where BUILD_TYPE is one of:
    sphix   -- generate an ReST file for the sphinx documentation
    master  -- generate a debian copyright file for determined-master
    agent   -- generate a debian copyright file for determined-agent
"""

import email
import os
import sys
from typing import IO, List, Optional

known_licenses = {
    "apache2": "Apache-2.0",
    "bsd2": "BSD 2-clause",
    "bsd3": "BSD 3-clause",
    "mit": "MIT",
    "mozilla": "Mozilla Public License",
    "unlicense": "Unlicense",
}


def indent(text: str, indent: str) -> str:
    return "\n".join(indent + line for line in text.splitlines())


def post_process(text: str) -> str:
    lines = text.splitlines()
    # Strip terminal white space on each line.
    lines = [line.rstrip() for line in lines]
    # Delete empty lines at the end of the file.
    while lines and not lines[-1]:
        lines.pop()
    return "\n".join(lines)


class License:
    def __init__(
        self,
        tag: str,
        text: str,
        name: Optional[str] = None,
        type: Optional[str] = None,
        master: str = "false",
        agent: str = "false",
        webui: str = "false",
    ) -> None:
        assert text.strip(), "License text not found"
        assert name is not None, "a Name header is required"
        assert type is not None, "a Type header is required"
        assert (
            type.lower() in known_licenses
        ), f"Type header must be one of {known_licenses}"
        assert master.lower() in {"true", "false"}, "Master must be true or false"
        assert agent.lower() in {"true", "false"}, "Agent must be true or false"
        assert webui.lower() in {"true", "false"}, "Webui must be true or false"

        self.tag = tag
        self.text = text
        self.name = name
        self.type = type.lower()
        self.master = master.lower() == "true"
        self.agent = agent.lower() == "true"
        self.webui = webui.lower() == "true"

    @classmethod
    def from_file(cls, tag: str, f: IO[str]) -> "License":
        # Licenses are saved as a simple rfc822 format (email headers + body).
        msg = email.message_from_file(f)
        meta = {x[0].lower(): x[1] for x in msg._headers}  # type: ignore
        text = msg.get_payload()
        return cls(tag, text, **meta)

    def sphinx_ref(self) -> str:
        """
        Example output:
            :ref:`BSD 3-clause <tomb>`
        """
        return f":ref:`{known_licenses[self.type]} <{self.tag}>`"

    def sphinx_entry(self) -> str:
        return "\n".join(
            [
                f".. _{self.tag}:",
                "",
                sphinx_format_header(self.name, "*"),
                "",
                ".. code::",
                "",
                indent(self.text, "   "),
            ]
        )

    def ascii_entry(self) -> str:
        return "\n".join(
            [
                f"{self.name}",
                "",
                indent(self.text, "    "),
            ]
        )


def sphinx_format_header(text: str, char: str) -> str:
    """
    Example output:

        *******
         WebUI
        *******
    """
    return "\n".join(
        [
            char * (len(text) + 2),
            f" {text}",
            char * (len(text) + 2),
        ]
    )


def gen_sphinx_table(licenses: List[License]) -> str:
    """
    Example output:

        .. list-table::
           :header-rows: 1

           * - Package
             - License
           * - gopkg.in/tomb.v1
             - :ref:`BSD 3-clause <tomb>`
    """

    lines = [
        ".. list-table::",
        "   :header-rows: 1",
        "",
        "   * - Package",
        "     - License",
    ]

    for license in licenses:
        lines.append(f"   * - {license.name}")
        lines.append(f"     - {license.sphinx_ref()}")

    return "\n".join(lines)


sphinx_preamble = """
######################
 Open Source Licenses
######################

The following sets forth attribution notices for third-party software
that may be contained in Determined. We thank the open-source community
for all of their contributions.
""".strip()


def build_sphinx(licenses: List[License]) -> str:
    """Build the sphinx-format attributions.txt with all attributions."""

    paragraphs = [
        sphinx_preamble,
        sphinx_format_header("WebUI", "*"),
        gen_sphinx_table([license for license in licenses if license.webui]),
        sphinx_format_header("Determined Master", "*"),
        gen_sphinx_table([license for license in licenses if license.master]),
        sphinx_format_header("Determined Agent", "*"),
        gen_sphinx_table([license for license in licenses if license.agent]),
    ]

    for license in licenses:
        paragraphs.append(license.sphinx_entry())

    return "\n\n".join(paragraphs)


def build_ascii(licenses: List[License], our_license_path: str) -> str:
    with open(our_license_path) as f:
        our_license_text = f.read()

    paragraphs = [
        our_license_text.rstrip(),
        "This software is bundled with each of the following projects, in part or in whole:",
    ]

    for license in licenses:
        paragraphs.append(license.ascii_entry())

    return "\n\n".join(paragraphs)


def read_dir(path: str) -> List[License]:
    licenses = []
    for name in os.listdir(path):
        lpath = os.path.join(path, name)
        with open(lpath) as f:
            try:
                licenses.append(License.from_file(name, f))
            except AssertionError:
                print(f"Error reading license file {lpath}", file=sys.stderr)
                raise
    licenses.sort(key=lambda license: license.name)
    return licenses


def main(build_type: str) -> int:
    if build_type not in ("master", "agent", "sphinx"):
        print(__doc__, file=sys.stderr)
        return 1

    licenses = read_dir(os.path.join(os.path.dirname(__file__), "licenses"))
    our_license_path = os.path.join(os.path.dirname(__file__), "..", "..", "LICENSE")

    if sys.argv[1] == "sphinx":
        gen = build_sphinx(licenses)
    elif sys.argv[1] == "master":
        gen = build_ascii(
            [license for license in licenses if license.master or license.webui],
            our_license_path,
        )
    elif sys.argv[1] == "agent":
        gen = build_ascii(
            [license for license in licenses if license.agent], our_license_path
        )

    gen = post_process(gen)

    print(gen)

    return 0


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        sys.exit(1)

    build_type = sys.argv[1]

    sys.exit(main(build_type))
