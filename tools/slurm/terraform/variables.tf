variable "project" {
  type    = string
  default = "determined-ai"
}

variable "region" {
  type    = string
  default = "us-west1"
}

# Note: The default value for this variable is extracted during some
# CircleCI workflows. Modify with caution.
variable "zone" {
  type    = string
  default = "us-west1-b"
}

variable "ssh_user" {
  type = string
}

variable "ssh_key_pub" {
  type = string
}

variable "ssh_allow_ip" {
  type = string
}

variable "name" {
  type = string
}

variable "vmLifetimeSeconds"{
  type = number
  default = 7200
}

variable "machine_type" {
  type    = string
  default = "n1-standard-8"
}

variable "boot_disk" {
  type    = string
  default = "projects/determined-ai/global/images/det-environments-slurm-ci-1687962155"
}
