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

variable "workload_manager" {
  type = string
  default = "slurm"
}

variable "vmLifetimeSeconds"{
  type = number
  default = 7200
}

variable "machine_type" {
  type    = string
  default = "n1-standard-8"
}

variable "gpu_enabled" {
  type = bool
  default = false
}

variable "gpus" {
  description = "GPU information. Type and count of GPU to attach to the instance template. See https://cloud.google.com/compute/docs/gpus more details"
  type = object({
    type  = string
    count = number
  })
  default = null
}


variable "allow_stopping_for_update" {
  type    = bool
  default = false
}

variable "boot_disk" {
  type    = string
}
