Determined AI Scalability Testing
=================================
This directory contains the Determined AI Scalability Testing (`daist`) python project.

The following execution entry points are provided:

- `./build_pkg.py`: Build the `daist` pip package. The output will be `./__dist__`.
- `./make_venv.py`: Create a `daist` virtual environment with the necessary dependencies installed 
  within. The location of the virtual environment will be `__venv__`. 
- `./run.py`: The main execution entry point to running `daist` tests against a `determined` 
  cluster under test. A virtual environment will be created as necessary. The command line 
  arguments are passed on to the python `unittest` framework, the underlying framework 
  used by `daist` to execute tests.
- `./uts.py`: The unit testing execution entry points. The associated unit tests are found in 
  the `uts` directory. All command line options are passed to the underlying python `unittest` 
  framework.

Examples of running tests:

- `./run.py uts.rest_api.test_locust.TestRO`: Run the rest API read only test
- `./run.py uts.metrics.test_latency.Test.`: Run the metric latency test.

Environment and Configuration
-----------------------------
`daist` supports configuration via a file and via environment variables. The environment 
variables take precedence. See `daist.models.environment` for an enumeration of supported 
environment variables.

By default, the configuration file found at `daist/config.d/config.conf` will be used. However, 
this can be changed by setting the `DAIST_CONFIG` environment variable.

Results
-------
By default, results are output to `daist/results/<hostname>/<ISO 8061 UTC timestamp>/`. This can 
be configured via `config.conf[exec][output]`. The manifest for the results is found in a 
`Result-<tags>.json` file. 

The `daist.models.base` module defines the base classes for data `daist` serialization. Artifacts 
associated with `daist` serialization classes take the following form: 

    <classname>-<zero or more tags>-<ISO 8061 UTC timestamp>.<extension>

    where
    
    <classname>: matches the name of the class that should be used for 
                 serialization/deserialization.
    <zero or more tags>: A variable number of tags are found here.
    <ISO 8061 UTC timestamp>: example - 2024-07-29T18-49-42Z
    <extension>: A extension that will indicate the file type, for example: .json, .yaml, .txt, .png
 
The `daist` log will be `test_run.log` within the associated results' directory.

The session configuration file is saved to the associated results' directory.
