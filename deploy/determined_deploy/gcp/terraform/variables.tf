// AUTH
variable "keypath" {
  type = string
}

// GCP

variable "identifier" {
  type = string
}

variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "zone" {
  type = string
  default = null
}

variable "network" {
  type = string
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

variable "create_database" {
  type = bool
  default = true
}


// CLUSTER

variable "master_instance_type" {
  type = string
  default = "n1-standard-16"
}

variable "agent_instance_type" {
  type = string
  default = "n1-standard-32"
}

variable "gpu_type" {
  type = string
  default = "nvidia-tesla-v100"
}

variable "gpu_num" {
  type = number
  default = 8
}

variable "max_instances" {
  type = number
  default = 8
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
  default = "5m"
}


// DETERMINED

variable "environment_image" {
  type = string
}

variable "det_version" {
  type = string
}


// MASTER

variable "instance_name" {
  type = string
  default = "det-master"
}

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

variable "db_instance_name" {
  type = string
  default = "det-postgres-db"
}

variable "db_name" {
  type = string
  default = "determined"
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
