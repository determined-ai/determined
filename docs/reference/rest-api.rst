.. _rest-api:
.. _rest-api-reference:

############
 REST API
############

Determined's REST APIs provide a way for users and external tools to interact with a Determined
cluster programmatically. Determined includes detailed documentation about all of the REST endpoints
provided by the API, alongside a playground for interacting with the API.

We use `protobuf <https://developers.google.com/protocol-buffers>`_ to define language-agnostic
message structures. We use these type definitions with `gRPC-gateway
<https://grpc-ecosystem.github.io/grpc-gateway/>`_ to provide consistent REST endpoints to serve
various needs.

Using these two tools we also auto-generate OpenAPI v2 spec (aka Swagger RESTful API Documentation
Specification) which lets us inline documentation for each endpoint and response in our codebase.

Having this specification prepared we can serve it to different tools to generate code for different
languages (eg swagger codegen) as well as provide web-based explorers into our APIs (e.g., Swagger
UI).

****************
 Authentication
****************

Most of the API calls to Determined cluster need to be authenticated. On each API call, the server
expects a Bearer token to be present.

To receive a token POST a valid username and password combination to the login endpoint, currently
at ``/api/v1/auth/login`` with the following format:

.. code:: json

   {
     "username": "string",
     "password": "string"
   }

Sample request:

.. code:: bash

   curl -s "${DET_MASTER}/api/v1/auth/login" \
     -H 'Content-Type: application/json' \
     --data-binary '{"username":"determined","password":""}'

Sample response:

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

Once we have the token, we should store it and attach it to future API calls under the
``Authorization`` header using the following format: ``Bearer $TOKEN``.

*********
 Example
*********

In this example, we will show how to use the REST APIs to unarchive an experiment that was
previously archived.

By looking at the archive endpoint entry from our API docs (Swagger UI), we see that all we need is
an experiment ID.

To find an experiment that was archived, we lookup the experiments endpoint to figure out which
filtering options we have. We see that we have ``archived`` and ``limit`` to work with. Using a
bearer token we authenticate our request. We then use the ``archived`` and ``limit`` query
parameters to limit the result set to only show a single archived experiment.

.. code:: bash

   curl -H "Authorization: Bearer ${token}" "${DET_MASTER}/api/v1/experiments?archived=true&limit=1"

Here's what our example response looks like. We see it matches the expected response shape.

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

Now that we have our desired experiment's ID, we use it to target the experiment through the
unarchive endpoint using a POST request as specified by the endpoint:

.. code:: bash

   curl -H "Authorization: Bearer ${token}" -X POST "${DET_MASTER}/api/v1/experiments/16/unarchive"
