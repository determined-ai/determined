#!/usr/bin/env python
import json
import re
import subprocess
import sys
import time
import urllib.parse
from collections import namedtuple

# First, you'll need to install `hub`: https://github.com/github/hub
#
# Make sure to use it once manually (i.e. `hub clone determined-ai/determined delme` to go through
# its token auth process.

# From your personal fork of the Determined repo, assuming you've named Determined "upstream":
# `get_jira_for_release.py <commit of last release> (<optionally most recent commit>)`
# e.g. `get_jira_for_release.py v0.10.9` when releasing v0.10.10
# This will give you the Jira query to tag releases for, and the TSV to copy into the release party
# spreadsheet.


JIRA_URL = "https://determinedai.atlassian.net/issues/?jql="


PullRequest = namedtuple("PullRequest", ["author", "title", "url", "commit"])

PRS_QUERY = """
query($endCursor: String) {
  repository(owner: "determined-ai", name: "determined") {
    pullRequests(
      states: [MERGED],
      first: 50,
      after: $endCursor,
      orderBy: {field: UPDATED_AT, direction: DESC}
    ) {
      nodes {
        author { login }
        title
        url
        updatedAt
        mergeCommit { oid }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}
"""


def setup():
    subprocess.run(["git", "fetch", "upstream", "--tags"], check=True)


def copy_to_clipboard(str):
    for args in [["pbcopy"], ["xsel", "-ib"]]:
        try:
            subprocess.run(args, check=True, input=str, universal_newlines=True)
            return
        except FileNotFoundError:
            pass
    raise Exception("No suitable clipboard program found")


def get_commits_in_range(start_commit, end_commit):
    proc = subprocess.run(
        ["git", "log", "--format=%H", "%s..%s" % (start_commit, end_commit)],
        check=True,
        stdout=subprocess.PIPE,
        universal_newlines=True,
    )
    return set(proc.stdout.splitlines())


def get_prs_with_commits(start_time=None):
    # Default to checking the past three weeks to cover the time since the last release and more.
    if start_time is None:
        start_time = time.time() - 21 * 86400
    prs = []
    with subprocess.Popen(
        ["hub", "api", "--paginate", "graphql", "-f", "query=" + PRS_QUERY], stdout=subprocess.PIPE,
    ) as proc:
        for line in proc.stdout:
            j = json.loads(line)
            prs += j["data"]["repository"]["pullRequests"]["nodes"]

            oldest_time = time.mktime(time.strptime(prs[-1]["updatedAt"], "%Y-%m-%dT%H:%M:%SZ"))
            if oldest_time < start_time:
                break

    prs = [
        PullRequest(
            author=pr["author"]["login"],
            title=pr["title"],
            url=pr["url"],
            commit=pr["mergeCommit"]["oid"],
        )
        for pr in prs
    ]
    prs.sort(key=lambda pr: pr.url, reverse=True)
    return prs


def copy_list_of_tickets(prs):
    pr_titles = [pr.title for pr in prs]
    all_tickets = [re.findall(r"(DET-\d+)", ti, flags=re.IGNORECASE) for ti in pr_titles]
    flattened = [item.upper() for sublist in all_tickets for item in sublist]
    # TODO: Use the Jira API for this.
    print("Open to bulk edit issues with fix version (also copied to clipboard):")
    jira_query = "issuekey in (%s)" % ", ".join(['"' + str(issue) + '"' for issue in flattened])
    search_url = JIRA_URL + urllib.parse.quote(jira_query)
    print(search_url)
    copy_to_clipboard(search_url)


def copy_release_party_tsv(prs):
    print("Copy this into the release party spreadsheet (also copied to clipboard):")
    release_tsv = "\n".join(
        [f"{pr.author}\tPR\t{pr.title}\t\t\t{pr.url}\t{pr.commit}" for pr in prs]
    )
    print(release_tsv)
    copy_to_clipboard(release_tsv)


def main(args):
    start_commit = args[1]
    end_commit = args[2] if len(args) == 3 else "HEAD"

    setup()

    all_commit_shas = get_commits_in_range(start_commit, end_commit)
    prs_with_shas = get_prs_with_commits()

    all_prs_in_commit_range = [pr for pr in prs_with_shas if pr.commit in all_commit_shas]

    copy_list_of_tickets(all_prs_in_commit_range)

    print("Press Enter when ready for the release party information.")
    input()

    copy_release_party_tsv(all_prs_in_commit_range)


if __name__ == "__main__":
    main(sys.argv)
