terraform {
  required_version = "~> 1.0"

  backend "s3" {
    # defined in backend_*.conf
  }

  required_providers {
    aws = {
      version = "~> 5.0"
    }
    null = {
      version = "~> 3.2"
    }
  }
}

provider "aws" {
  region = "us-west-2"
  default_tags {
    tags = {
      owner        = "determined_ci"
      gh_team         = "docs-team"
      long_running = "docs_site"
    }
  }
}

provider "null" {
}
