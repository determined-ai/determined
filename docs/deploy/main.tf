# module "main-docsite-infra" {
#   source = "./docsite-infra"
#
#   # vars go here
#   det_version = ""
# }

module "test-docsite-infra" {
  source = "./docsite-infra"

  # vars go here
  det_version = ""
}

terraform {
  required_version = "~> 1.0"

  backend "s3" {
    bucket = "determined-ai-docs-terraform"
    key    = "terraform.tfstate"
    region = "us-west-2"
  }
}

provider "aws" {
  version = "~> 4.63.0"
  region  = "us-west-2"
}

provider "null" {
  version = "~> 3.2.1"
}
