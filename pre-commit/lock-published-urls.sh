#!/bin/sh

# redirects.py inspects files to detect missing redirects, so generated files
# must be generated before the publish step.
make -C docs attributions.rst

# Generate redirects.
python3 docs/redirects.py publish

# If docs/.redirects/all_published_urls_ever.json has been modified locally,
# then `redirects.py publish` has made modifications that we need to commit.
git diff --exit-code -- docs/.redirects/all_published_urls_ever.json
exit $?
