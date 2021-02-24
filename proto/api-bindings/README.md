# Determined API Bindings

To generate the bindings, place the Swagger specifications file
(aka `api.swagger.json`) for a desired version of Determined in the project directory.
You can find this file at `DETERMINED_CLUSTER_ADDRESS/api/v1/api.swagger.json`.
Once you have the spec file in the right place simply call `make` to generate
API bindings for a subset of the supported languages.
To generate code for a specific language, eg X, issue: `make get-deps build-X`.
