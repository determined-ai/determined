from __future__ import print_function
from typing import Any, List
from argparse import Namespace
from determined.common.declarative_argparse import Arg, Cmd
import determined.common.api.swagger as swg
import yaml

@swg.auth_required
def list(_: Namespace) -> None:
    api_response = swg.job_api.determined_get_jobs(resource_pools=["default"]) # TODO type info for (optional) input and output
    print(yaml.dump(api_response))

args_description = [
    Cmd("j|ob", None, "manage job", [
        Cmd("list", list, "list jobs", [
            Arg("-o", "--output", type=str, default="yaml",
                help="Output format, one of json|yaml")
        ]),
    ])
]  # type: List[Any]
