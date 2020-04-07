GraphQL development
===================

Metadata files and generated code
---------------------------------

There are several sets of files related to the GraphQL schema:

``master/static/hasura-metadata.json``
   A JSON description of the GraphQL schema in Hasura's native format.
   This is, ultimately, the source of truth about what Hasura should be
   serving. It should be updated in a development environment by working
   with Hasura's web console or running the scripts in
   ``scripts/hasura``, then running ``make graphql-schema``. The master
   sends this file to Hasura on startup to ensure that it is always
   serving an up-to-date schema.

``master/graphql-schema.json``
   A JSON description of the GraphQL schema in a standard format. This
   file can be generated from the one above through Hasura, but we track
   it so that we can generate code for the schema while bootstrapping
   without having to have a server running.

``common/determined_common/api/gql.py``
   Python code generated based on the schema. Although it is rather
   large and can be generated from the JSON schema description, we track
   it to avoid having to deal with installing and running extra
   dependencies early in the process in the CI environment (which
   includes tests on Windows).

``webui/elm/src/DetQL/**/*.elm``
   Elm code generated based on the schema. The situation is the same as
   for the Python code, though there are many files and they are even
   larger.

The relevant Make targets are:

``graphql-schema``
   Connects to Hasura and updates the JSON schema description files.

``graphql-elm`` and ``graphql-python``
   Generate Elm and Python code based on the standard schema description
   file (does not require Hasura to be running).

``graphql``
   Runs ``graphql-schema`` and then the language-specific targets.

.. note::
   After you update the GraphQL schema by doing something in Hasura's
   web console or updating the database schema, running ``make graphql``
   will regenerate everything that needs to be updated. You can then
   review and commit the changes.

Hasura management scripts
-------------------------

``scripts/hasura/export-metadata.sh``
   This dumps the current Hasura schema to the proper location in the
   repo. ``make graphql`` will run and then regenerate the code as
   necessary, so it should not usually be necessary to use this
   directly.

``scripts/hasura/import-metadata.sh``
   This updates the Hasura schema from the file in the repo. The master
   will do the same on startup, so it should not usually be necessary to
   use this directly.

``scripts/hasura/reload-metadata.sh``
   This tells Hasura to reread the current database schema. The master
   will normally take care of updating the database and Hasura schemata
   together, so it should not usually be necessary to use this directly.

``scripts/hasura/run.sh``
   This runs Hasura by itself with an appropriate configuration.
   ``det-deploy`` and the other methods of managing the Determined services
   take Hasura into account, so this is mainly a convenient helper for
   if you're doing something different.
