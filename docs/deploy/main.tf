terraform {
  required_version = "~> 1.0"

  backend "s3" {
    bucket = "hpe-mlde-docs-terraform"
    key    = "terraform.tfstate"
    region = "us-west-2"
  }
}

provider "aws" {
  version = "~> 2.19.0"
  region  = "us-west-2"
}

provider "null" {
  version = "~> 2.1.2"
}
