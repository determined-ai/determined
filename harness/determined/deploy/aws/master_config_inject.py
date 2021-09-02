#!/usr/bin/env python
#
# Replaces MasterConfigTemplate in all CloudFormation files with the content of
# `injector/master.yaml.tmpl`.
# Also see harness/Makefile target `aws-master-config-inject`.
#
# We reuse the same master template in a number of CloudFormation templates.
# This small tool allows to have one single source of truth for the master
# config template.
# If you need to change the template, edit `injector/master.yaml.tmpl`,
# then run this tool to update all the CF files.

import re
from itertools import chain
from pathlib import Path
from typing import Dict

START_MARKER = re.compile(r"INJECT CODE: (.+)")
END_MARKER = re.compile("END INJECT CODE")


def template_rewrite(template_path: Path, context: Dict[str, str]) -> None:
    with template_path.open("r") as fin:
        temp_path = Path(str(template_path) + ".temp")
        with temp_path.open("w") as fout:
            matching = False
            for line in fin:
                if matching:
                    if END_MARKER.search(line):
                        matching = False
                else:
                    m = START_MARKER.search(line)
                    if m:
                        matching = True
                        key = m.group(1)
                        fout.write(line)
                        fout.write(context[key])
                        continue
                if not matching:
                    fout.write(line)
        temp_path.replace(template_path)


def _indent_line(line: str, n: int) -> str:
    return (" " * n + line) if line != "\n" else line


def inject_master_config(target_path: Path, content_path: Path, indent: int) -> None:
    CONTENT_KEY = "MasterConfigTemplate"
    with content_path.open("r") as fin:
        context = {CONTENT_KEY: "".join(_indent_line(line, indent) for line in chain(["|\n"], fin))}
    template_rewrite(target_path, context)


if __name__ == "__main__":
    deploy_aws_dir = Path(__file__).parent.resolve()
    templates_dir = deploy_aws_dir / "templates"
    content_path = deploy_aws_dir / "injector" / "master.yaml.tmpl"
    for target_path in templates_dir.glob("*.yaml"):
        inject_master_config(target_path, content_path, 6)
