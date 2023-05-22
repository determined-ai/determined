#!/bin/bash
set -eux
exec 2>&1

case "$PR_TITLE" in
    fix:* | feat:*)
        echo "fix or feat, adding to release list"
        ;;
    [a-z]*:*)
        echo "not fix or feat, skipping"
        exit
        ;;
    *)
        echo "unknown, adding to release list to be safe"
        ;;
esac

# Fetch various IDs.
repo_id="$(gh api graphql -f query='
query {
  repository(owner:"determined-ai", name:"release-party-issues") {
    id
  }
}
' --jq '.data.repository.id')"

project_id="$(gh api graphql -f query='
query {
  organization(login: "determined-ai") {
    projectsV2(query: "Next release", first: 100) {
      nodes {
        id
        title
      }
    }
  }
}
' --jq '.data.organization.projectsV2.nodes | map(select(.title=="Next release")) | .[0].id')"

status_json="$(gh api graphql -f query='
query($project: ID!) {
  node(id: $project) {
    ... on ProjectV2 {
      field(name: "Status") {
        ... on ProjectV2SingleSelectField {
          id
          options {
            id
            name
          }
        }
      }
    }
  }
}
' -f project="$project_id" --jq '.data.node.field')"

status_id="$(jq .id <<<"$status_json")"
needs_testing_id="$(jq -r '.options | map(select(.name == "Needs testing")) | .[0].id' <<<"$status_json")"

# Create new issue.
issue_id="$(gh api graphql -f query='
mutation($repo: ID!, $title: String!, $body: String!) {
  createIssue(input: {repositoryId: $repo, title: $title, body: $body}) {
    issue {
      id
    }
  }
}
' -f repo="$repo_id" -f title="Test $PR_REPO#$PR_NUM ($PR_TITLE)" -f body="$PR_URL" --jq '.data.createIssue.issue.id')"

# Add issue to project.
item_id="$(gh api graphql -f query='
mutation($project: ID!, $item: ID!) {
  addProjectV2ItemById(input: {projectId: $project, contentId: $item}) {
    item {
      id
    }
  }
}
' -f project="$project_id" -f item="$issue_id" --jq '.data.addProjectV2ItemById.item.id')"

# Set status of item in project.
gh api graphql -f query='
mutation($project: ID!, $item: ID!, $field: ID!, $value: String) {
  updateProjectV2ItemFieldValue(input: {projectId: $project, itemId: $item, fieldId: $field, value: {singleSelectOptionId: $value}}) {
    projectV2Item {
     id
   }
  }
}
' -f project="$project_id" -f item="$item_id" -f field="$status_id" -f value="$needs_testing_id"
