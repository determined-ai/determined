from __future__ import print_function
from typing import Any, List
from argparse import Namespace
from determined.common.declarative_argparse import Arg, Cmd
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
def list(_: Namespace) -> None:
    api_response = fapi.sync_apis.jobs_api.determined_get_jobs(resource_pools=["default"])
    print(api_response.jobs)


args_description = [
    Cmd("j|ob", None, "manage job", [
        Cmd("list", list, "list jobs", [
            Arg("-o", "--output", type=str, default="yaml",
                help="Output format, one of json|yaml"),
        ]),
    ])
]  # type: List[Any]
