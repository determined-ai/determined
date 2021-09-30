from __future__ import print_function
from typing import Any, List, Dict, Union
from argparse import Namespace
from determined.common.declarative_argparse import Arg, Cmd
from determined.cli import  render
import yaml
import json
import determined.common.api.fapi as fapi
# from determined.common.api.fastapi_client.models import V1LoginRequest

# import determined.common.api.swagger as swg
# import determined.common.api.swagger_client.models as models
# @swg.auth_required
# def list(_: Namespace) -> None:
#     # type info shows for some language servers but not all since the generated code
#     # is using legacy docstring types
#     api_response = swg.job_api.determined_get_jobs(resource_pools=["default"])
#     print(api_response.jobs)


@fapi.auth_required
def list(args: Namespace) -> None:
    response = fapi.sync_apis.jobs_api.determined_get_jobs(resource_pools=[args.resource_pool])
    if response.jobs is None: # TODO mark the proto as required.
        return
    jobs_json = fapi.to_json(response.jobs)
    if args.output == 'yaml':
        print(yaml.safe_dump(jobs_json, default_flow_style=False))
    elif args.output == 'json':
        print(json.dumps(jobs_json, indent=4, default=str))
    elif ['csv', 'table'].count(args.output) > 0:
        # render.tabulate_or_csv # TODO maybe add support for csv or tabular format. ref exp list
        raise NotImplementedError(f"Output not implemented, adopt a cat to unlock: {args.output}")
    else:
        raise ValueError(f"Bad output format: {args.output}")


args_description = [
    Cmd("j|ob", None, "manage job", [
        Cmd("list", list, "list jobs", [
            Arg("-o", "--output", type=str, default="yaml",
                help="Output format, one of json|yaml"),
            Arg("-rp", "--resource-pool", type=str, default="default",
                help=""),
        ]),
    ])
]  # type: List[Any]
