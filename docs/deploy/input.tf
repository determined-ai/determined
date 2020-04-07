variable "det_version" {
  type        = "string"
  description = "The Determined version the docs are currently built under"
}

variable "build_dir" {
  type        = "string"
  description = "The location where the built docs can be found"
}

locals {
  domain = "docs.determined.ai"
}
