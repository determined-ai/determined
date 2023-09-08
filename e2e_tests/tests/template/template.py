import os
import re
import subprocess

from determined.common import api
from tests import detproc


def set_template(sess: api.Session, template_name: str, template_file: str) -> str:
    completed_process = maybe_set_template(sess, template_name, template_file)
    assert completed_process.returncode == 0
    m = re.search(r"Set template (\w+)", str(completed_process.stdout))
    assert m is not None
    return str(m.group(1))


def maybe_set_template(
    sess: api.Session, template_name: str, template_file: str
) -> subprocess.CompletedProcess:
    command = [
        "det",
        "template",
        "set",
        template_name,
        os.path.join(os.path.dirname(__file__), template_file),
    ]
    return detproc.run(sess, command, universal_newlines=True, stdout=subprocess.PIPE)


def describe_template(sess: api.Session, template_name: str) -> str:
    completed_process = maybe_describe_template(sess, template_name)
    assert completed_process.returncode == 0
    return str(completed_process.stdout)


def maybe_describe_template(sess: api.Session, template_name: str) -> subprocess.CompletedProcess:
    command = ["det", "template", "describe", template_name]
    return detproc.run(sess, command, universal_newlines=True, stdout=subprocess.PIPE)
