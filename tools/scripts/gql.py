import os
from typing import Any

import requests

GRAPHQL_URL = "https://api.github.com/graphql"

DEBUG = os.environ.get("DET_DEBUG") == "1"


class GraphQLQuery(str):
    def __call__(self, **args: Any) -> Any:
        if DEBUG:
            print("================ GraphQL query")
            print(self.strip())
            print(args)
        r = requests.post(
            GRAPHQL_URL,
            headers={"Authorization": "bearer " + os.environ["GITHUB_TOKEN"]},
            json={"query": self, "variables": args},
        )
        r.raise_for_status()
        j = r.json()
        if DEBUG:
            print(j)
        try:
            return j["data"]
        except Exception:
            print(j)
            raise


get_repo_id = GraphQLQuery(
    """
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id
  }
}
"""
)

get_pr_id = GraphQLQuery(
    """
query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      id
    }
  }
}
"""
)


search_projects = GraphQLQuery(
    """
query($owner: String!, $q: String!) {
  organization(login: $owner) {
    projectsV2(query: $q, first: 100) {
      nodes {
        id
        title
      }
    }
  }
}
"""
)

get_project_status_field = GraphQLQuery(
    """
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
"""
)

create_issue = GraphQLQuery(
    """
mutation($repo: ID!, $title: String!, $body: String!) {
  createIssue(input: {repositoryId: $repo, title: $title, body: $body}) {
    issue {
      id
    }
  }
}
"""
)

add_item_to_project = GraphQLQuery(
    """
mutation($project: ID!, $item: ID!) {
  addProjectV2ItemById(input: {projectId: $project, contentId: $item}) {
    item {
      id
    }
  }
}
"""
)

get_status_field_info = GraphQLQuery(
    """
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
"""
)

set_project_item_status = GraphQLQuery(
    """
mutation($project: ID!, $item: ID!, $field: ID!, $value: String) {
  updateProjectV2ItemFieldValue(
    input: {
      projectId: $project, itemId: $item, fieldId: $field, value: {singleSelectOptionId: $value}
    }
  ) {
    projectV2Item {
     id
   }
  }
}
"""
)

get_pr_labels = GraphQLQuery(
    """
query($id: ID!) {
  node(id: $id) {
    ... on PullRequest {
      labels(first: 100) {
        nodes {
          name
        }
      }
    }
  }
}
"""
)

get_pr_merge_commit_and_url = GraphQLQuery(
    """
query($id: ID!) {
  node(id: $id) {
    ... on PullRequest {
      url
      mergeCommit {
        oid
      }
    }
  }
}
"""
)

get_pr_info = GraphQLQuery(
    """
query($id: ID!) {
  node(id: $id) {
    ... on PullRequest {
      number
      title
      url
      body
      repository {
        owner {
          login
        }
        name
      }
    }
  }
}
"""
)


get_pr_title = GraphQLQuery(
    """
query($id: ID!) {
  node(id: $id) {
    ... on PullRequest {
      title
    }
  }
}
"""
)


get_pr_state = GraphQLQuery(
    """
query($id: ID!) {
  node(id: $id) {
    ... on PullRequest {
      state
    }
  }
}
"""
)


delete_project_item = GraphQLQuery(
    """
mutation($project: ID!, $item: ID!) {
  deleteProjectV2Item(input: {projectId: $project, itemId: $item}) {
    deletedItemId
  }
}
"""
)
