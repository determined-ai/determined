// AUTH
variable "keypath" {
  type = string
  default = null
}

// GCP

variable "cluster_id" {
  type = string
}

variable "project_id" {
  type = string
}

variable "network" {
  type = string
}

variable "region" {
  type = string
}

variable "zone" {
  type = string
  default = null
}

variable "subnetwork" {
  type = string
  default = null
}

variable "gcs_bucket" {
  type = string
  default = null
}

variable "service_account_email" {
  type = string
  default = null
}

variable "create_static_ip" {
  type = bool
  default = true
}


// CLUSTER

variable "master_instance_type" {
  type = string
  default = "n1-standard-2"
}

variable "agent_instance_type" {
  type = string
  default = "n1-standard-32"
}

variable "gpu_type" {
  type = string
  default = "nvidia-tesla-k80"
}

variable "gpu_num" {
  type = number
  default = 8
}

variable "max_dynamic_agents" {
  type = number
  default = 5
}

variable "static_agents" {
  type = number
  default = 0
}

variable "preemptible" {
  type = bool
  default = false
}

variable "agent_docker_network" {
  type = string
  default = "host"
}

variable "master_docker_network" {
  type = string
  default = "determined"
}

variable "max_idle_agent_period" {
  type = string
  default = "10m"
}

variable "min_cpu_platform_master" {
  type = string
  default = "Intel Skylake"
}

variable "min_cpu_platform_agent" {
  type = string
  default = "Intel Broadwell"
}

// DETERMINED

variable "environment_image" {
  type = string
}

variable "det_version" {
  type = string
}


// MASTER

variable "scheme" {
  type = string
  default = "http"
}

variable "port" {
  type = number
  default = 8080
}


// DATABASE

variable "db_version" {
  type = string
  default = "POSTGRES_11"
}

variable "db_tier" {
  type = string
  default = "db-f1-micro"
}

variable "db_username" {
  type = string
  default = "postgres"
}

variable "db_password" {
  type = string
  default = "postgres"
}

variable "hasura_secret" {
  type = string
  default = "secret"
}
