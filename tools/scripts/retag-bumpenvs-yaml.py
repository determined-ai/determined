"""
update-bumpenvs-yaml.py variant that doesn't read artifacts.
It will retag all images in bumpenvs.yaml based on provided tags.
The --release argument will drop the -dev in the image name for a new image.

For example, if tags a and b are provided, it will change the image names as such:
    previous: {new: repo-dev:a, old: repo-dev:x}
    next: {new: repo-dev:b, old: repo-dev:a}

If --release is provided, this change would look like the following:
    previous: {new: repo-dev:a, old: repo-dev:x}
    next: {new: repo:b, old: repo-dev:a}

Usage: python retag-bumpenvs-yaml.py path/to/bumpenvs.yaml OLD_TAG NEW_TAG [--release]
"""

import argparse
import pathlib

from ruamel import yaml

def run(old_tag: str, new_tag: str, yaml_path: str, release: bool) -> None:
    with open(yaml_path) as f:
        conf = yaml.YAML(typ="safe", pure=True).load(f)

    for image_type in conf:
        if old_tag not in conf[image_type]['new']:
            continue
        replace_image(conf[image_type], new_tag, release)

    with open(yaml_path, "w") as f:
        yaml.YAML(typ="safe", pure=True).dump(conf, f)

def replace_image(subconf: str, new_tag: str, release: bool) -> None:
    old_tag = subconf["new"].split(":")[-1]
    subconf["old"] = subconf["new"]
    if release:
        subconf["new"] = subconf["new"].replace("-dev:"+old_tag, ":"+new_tag)
    else:
        subconf["new"] = subconf["new"].replace(old_tag, new_tag)

if __name__== "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("yaml_path", type=pathlib.Path, help="path/to/bumpenvs.yaml")
    parser.add_argument("old_tag", type=str, help="image tag to replace")
    parser.add_argument("new_tag", type=str, help="new image tag")
    parser.add_argument("--release", action="store_true", help="drops -dev in repo name")
    args = parser.parse_args()

    run(args.old_tag, args.new_tag, args.yaml_path, args.release)

