import json
import os
import sys
import collections


def merge_dict(d1, d2):
    """
    Modifies d1 in-place to contain values from d2.  If any value
    in d1 is a dictionary (or dict-like), *and* the corresponding
    value in d2 is also a dictionary, then merge them in-place.
    """

    for k, v2 in d2.items():
        v1 = d1.get(k) # returns None if v1 has no value for this key
        if (isinstance(v1, collections.Mapping) and isinstance(v2, collections.Mapping)):
            merge_dict(v1, v2)
        else:
            d1[k] = v2


def capitalize(s: str) -> str:
    if len(s) <= 1:
        return s.title()
    return s[0].upper() + s[1:]


def clean(fn: str) -> None:
    with open(fn, "r") as fp:
        spec = json.load(fp)

    # Add tag descriptions.
    spec["tags"] = [
        {
            "name": "Authentication",
            "description": "Login and logout of the cluster",
        },
        {
            "name": "Users",
            "description": "Manage users",
        },
        {
            "name": "Cluster",
            "description": "Manage cluster components",
        },
        {
            "name": "Experiments",
            "description": "Manage experiments",
        },
        {
            "name": "Templates",
            "description": "Manage templates",
        },
        {
            "name": "Models",
            "description": "Manage models",
        }
    ]

    # Update path names to be consistent.
    paths = {}
    for key, value in spec["paths"].items():
        paths[key.replace(".", "_")] = value
    spec["paths"] = paths

    for key, value in spec["definitions"].items():
        # Remove definitions that should be hidden from the user.
        if key == "protobufAny":
            value["title"] = "Object"
        elif key == "protobufNullValue":
            value["title"] = "NullValue"

        # Clean up titles.
        if "title" not in value:
            value["title"] = "".join(capitalize(k) for k in key.split(sep="v1"))

    with open("swagger-patch.json", "r") as diff_f:
        diff = json.load(diff_f)
        merge_dict(spec, diff)

    with open(fn, "w") as fp:
        json.dump(spec, fp)


def main() -> None:
    files = []
    for r, d, f in os.walk(sys.argv[1]):
        for file in f:
            if file.endswith(".json"):
                clean(os.path.join(r, file))


if __name__ == '__main__':
    main()
