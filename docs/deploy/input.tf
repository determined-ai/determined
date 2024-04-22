variable "det_version" {
  type        = string
  description = "The Determined version the docs are currently built under"
}
variable "det_variant" {
  type        = string
  description = "The Determined variant the docs are currently built under"
  validation {
    condition = contains(["EE", "OSS"], var.det_variant)
    error_message = "The variant must be one of OSS or EE"
  }
}

locals {
  config = yamldecode(file("${path.module}/vars_${var.det_variant}.yaml"))
}
