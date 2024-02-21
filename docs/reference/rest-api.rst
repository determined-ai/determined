.. _rest-api:

.. _rest-api-reference:

##########
 REST API
##########

The Determined REST API provides a way to interact with a Determined cluster programmatically. The
API reference documentation includes detailed information about the REST API endpoints and a
playground for interacting with the API.

The `protobuf <https://protobuf.dev/>`_ mechanism is used to define language-agnostic message
structures. These type definitions are used with `gRPC-gateway
<https://grpc-ecosystem.github.io/grpc-gateway/>`_ to provide consistent REST endpoints that serve
various needs.

These tools are used to autogenerate an OpenAPI v2 specification, which inlines documentation for
each endpoint and response message. The specification can be served to different tools to generate
code for different languages and to provide web-based explorers, such as the Swagger UI, for the
API.

***********
 Reference
***********

+----------------------------------------------------+
| REST API Reference                                 |
+====================================================+
| `Determined REST API <../rest-api/index.html>`__   |
+----------------------------------------------------+

The REST API reference documentation lists available endpoints grouped by workflow. Click an
endpoint method to see the expected input parameters and response. You can also use **Try it out**
button to make an HTTP request against the endpoint. You need to have the appropriate cookie set and
a running cluster for an interactive request.

If you have access to a running Determined cluster you can try the live-interact version by clicking
the API icon from the Determined WebUI or by navigating to ``/docs/rest-api/`` on your Determined
cluster.

****************
 Authentication
****************

Most of the API calls to a Determined cluster require authentication. On each API call, the server
expects a Bearer token.

To receive a token, POST a valid username and password combination to the login endpoint,
``/api/v1/auth/login`` using the following format:

.. code:: json

   {
     "username": "string",
     "password": "string"
   }

Example request:

.. code:: bash

   curl -s "${DET_MASTER}/api/v1/auth/login" \
     -H 'Content-Type: application/json' \
     --data-binary '{"username":"determined","password":""}'

Example response:

.. code:: json

   {
     "token": "string",
     "user": {
       "username": "string",
       "admin": true,
       "active": true,
       "agent_user_group": {
         "agent_uid": 0,
         "agent_gid": 0
       }
     }
   }

When you receive the token, store it and attach it to future API calls under the ``Authorization``
header in the ``Bearer $TOKEN`` format.

*********
 Example
*********

This example shows how to use the REST API to unarchive a previously archived experiment.

To find an archived experiment, look up the ``experiment`` endpoint to find which filtering options
are provided. They are ``archived`` and ``limit``. Including a bearer token to authenticate the
request, use the ``archived`` and ``limit`` query parameters to limit the result set to only show a
single archived experiment:

.. code:: bash

   curl -H "Authorization: Bearer ${token}" "${DET_MASTER}/api/v1/experiments?archived=true&limit=1"

JSON response:

.. code:: json

   {
     "experiments": [
       {
         "id": 16,
         "description": "mnist_pytorch_const",
         "labels": [],
         "startTime": "2020-08-26T20:12:35.337160Z",
         "endTime": "2020-08-26T20:12:51.951720Z",
         "state": "STATE_COMPLETED",
         "archived": true,
         "numTrials": 1,
         "progress": 0,
         "username": "determined"
       }
     ],
     "pagination": {
       "offset": 0,
       "limit": 1,
       "startIndex": 0,
       "endIndex": 1,
       "total": 1
     }
   }

In the archive endpoint entry, you can see that all that you need is an experiment ID.

With the experiment ID you want, you can now unarchive the experiment using the ``unarchive``
endpoint in a POST request:

.. code:: bash

   curl -H "Authorization: Bearer ${token}" -X POST "${DET_MASTER}/api/v1/experiments/16/unarchive"
