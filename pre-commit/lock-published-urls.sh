#!/bin/sh

make -C docs lock-published-urls

# If docs/.redirects/all_published_urls_ever.json has been modified locally,
# then `redirects.py publish` has made modifications that we need to commit.
git diff --exit-code -- docs/.redirects/all_published_urls_ever.json
exit $?
