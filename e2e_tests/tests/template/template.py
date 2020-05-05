import os
import re
import subprocess

from tests import config as conf


def set_template(template_name: str, template_file: str) -> str:
    completed_process = maybe_set_template(template_name, template_file)
    assert completed_process.returncode == 0
    m = re.search(r"Set template (\w+)", str(completed_process.stdout))
    assert m is not None
    return str(m.group(1))


def maybe_set_template(template_name: str, template_file: str) -> subprocess.CompletedProcess:
    command = [
        "det",
        "-m",
        conf.make_master_url(),
        "template",
        "set",
        template_name,
        os.path.join(os.path.dirname(__file__), template_file),
    ]
    return subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE)


def describe_template(template_name: str) -> str:
    completed_process = maybe_describe_template(template_name)
    assert completed_process.returncode == 0
    return str(completed_process.stdout)


def maybe_describe_template(template_name: str) -> subprocess.CompletedProcess:
    command = ["det", "-m", conf.make_master_url(), "template", "describe", template_name]
    return subprocess.run(command, universal_newlines=True, stdout=subprocess.PIPE)
