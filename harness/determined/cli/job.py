from __future__ import print_function
from typing import Any, List
from argparse import Namespace
from determined.common.declarative_argparse import Arg, Cmd
import determined.common.api.swagger as swg


@swg.auth_required
def list(args: Namespace) -> None:
    api_response = swg.experiment_api.determined_get_experiment(1)
    print(api_response)
    pass

args_description = [
    Cmd("j|ob", None, "manage job", [
        Cmd("list", list, "list jobs", [
            Arg("-o", "--output", type=str, default="yaml",
                help="Output format, one of json|yaml")
        ]),
    ])
]  # type: List[Any]
