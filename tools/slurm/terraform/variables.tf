variable "project" {
  type    = string
  default = "determined-ai"
}

variable "region" {
  type    = string
  default = "us-west1"
}

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

variable "machine_type" {
  type    = string
  default = "n1-standard-8"
}

variable "boot_disk" {
  type    = string
  default = "projects/determined-ai/global/images/det-environments-slurm-ci-1686220344"
}
