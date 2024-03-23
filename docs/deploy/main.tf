terraform {
  required_version = "~> 1.0"

  backend "s3" {
    bucket = "determined-ai-docs-terraform"
    key    = "terraform.tfstate"
    region = "us-west-2"
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
      team         = "docs-team"
      long_running = "docs_site"
    }
  }
}

provider "null" {
}
