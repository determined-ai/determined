# tools/scripts

This directory is the home for tools that directly assist in the development of
the determined repository.

## `bumpenvs`: How to bump task environment versions

Our task environments are versioned separately from the main determined
repository. The motivation for this is that customers who maintain custom
Docker images as extensions to our image should not have to rebuild their
custom images as often as we cut releases. Since in practice our prebuilt
environments are pretty slow to change, and since some organizations have a
long and tedious security review for task environments, this is a good thing
for customers. But it does mean that updates to the environments repo have to
be registered in the determined repo. Here is the process:

1. Land the desired change in the environments repo. Remember the full commit
   hash (we'll call it `THECOMMIT` in these steps).

2. Wait for the post-merge-to-master CircleCI jobs on the environments repo to
   finish. These will publish the relevant Docker/AWS/GCP images and create
   machine-readable artifacts containing the image tags.

3. Enter the `tools/scripts` directory of the determined repo.

4. Run `./update-bumpenvs-yaml.py bumpenvs.yaml THECOMMIT`. This will fetch the
   above-mentioned machine-readable artifacts from the CircleCI jobs of the
   environments repository, parse out the image tags, and update the relevant
   entries in `bumpenvs.yaml`.  For every artifact found, this will set that
   artifact's `old` value to the previous `new` value, and set the new `new`
   value to the artifact produced by CI, including the task environments and
   the agent AMIs.

5. (optional) Run `./refresh-ubuntu-amis.py bumpenvs.yaml`.  This will fetch
   the up-to-date Ubuntu AMIs for each region for each of the `*_master_ami`
   and `*_bastion_ami` image tags in bumpenvs.yaml.  This isn't strictly
   necessary; we just need to run it periodically, and now is a fine time.

6. Run `./bumpenvs.py bumpenvs.yaml`.  This will do a simple string replacement
   in the repository, replacing the `old` values with the `new` values for
   every image type identified in `bumpenvs.yaml`.

## `gen-attributions.py`: Automated OSS Compliance

OSS License compliance is is important for being good OSS citizens and it is
very important for some of our strategic partnerships.  There are several such
documents that we must maintain, and because golang binaries and the WebUI are
effectively statically linked, the maintenance effort is potentially very high.

Therefore, to make life a little bit easier, we have a system for generating
those license documents.  All you have to do is create specially formatted
license files for every external dependency of `determined-master`,
`determined-agent`, or the WebUI, and place them in the `licenses/` directory.

The format of the file is rfc822 format (the email format), which is a set of
headers, followed by an empty line, followed by some preformatted text.  The
headers contain metadata about the license and everything after the empty line
should be the exact copy/pasted license text from the project in question.
Here is an example:

    Name: github.com/lib/pq
    Type: mit
    Agent: false
    Master: true
    Webui: false

    Copyright (c) 2011-2013, 'pq' Contributors
    Portions Copyright (C) 2011 Blake Mizerany

    Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

The relevant metadata are:

* `Name`: the full name of the dependency
* `Type`: one of `apache2`, `bsd2`, `bsd3`, `mit`, `mozilla`, or `unlicense`
* `Agent`: boolean indicating if this is a dependency of `determined-agent`
* `Master`: boolean indicating if this is a dependency of `determined-master`
* `Webui`: boolean indicating if this is a dependency of the WebUI

For instructions on how to use `gen-attributions.py`, just run it with no arguments.

## `lock-api-state.sh`: Lock-in current API state

We use [buf.build](https://docs.buf.build/) to provide backward compatibility check
for API changes in Determined. By running this script on each release, we store a
snapshot of the API state and on each following change `buf` compares the new state
with the old to ensure that backward compatibility is preserved.
