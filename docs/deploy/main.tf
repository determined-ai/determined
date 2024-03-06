terraform {
  required_version = "~> 1.0"

  backend "s3" {
    bucket = "determined-ai-docs-terraform"
    key    = "terraform.tfstate"
    region = "us-west-2"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 2.19.0"
    }
    null = {
      source  = "hashicorp/null"
      version = ">= 2.1.2"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}
