/******************************************
	AUTH
 *****************************************/

variable "keypath" {
  type = string
  default = null
}

/******************************************
	GCP
 *****************************************/

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

variable "no_filestore" {
  type = bool
  default = false
}

variable "filestore_address" {
  type = string
  default = ""
}

variable "gcs_bucket" {
  type = string
  default = null
  description = "The name for the provided GCS bucket"
}

variable "service_account_email" {
  type = string
  default = null
}

variable "create_static_ip" {
  type = bool
  default = true
}


/******************************************
	Cluster
 *****************************************/

variable "master_instance_type" {
  type = string
  default = "n1-standard-2"
}

variable "aux_agent_instance_type" {
  type = string
  default = "n1-standard-4"
}

variable "compute_agent_instance_type" {
  type = string
  default = "n1-standard-32"
}

variable "gpu_type" {
  type = string
  default = "nvidia-tesla-t4"
}

variable "gpu_num" {
  type = number
  default = 4
}

variable "min_dynamic_agents" {
  type = number
  default = 0
}

variable "max_dynamic_agents" {
  type = number
  default = 5
}

variable "preemptible" {
  type = bool
  default = false
}

variable "operation_timeout_period" {
  type = string
  default = "5m"
}

variable "agent_docker_network" {
  type = string
  default = "host"
}

variable "master_docker_network" {
  type = string
  default = "determined"
}

variable "max_aux_containers_per_agent" {
  type = number
  default = 100
}

variable "max_idle_agent_period" {
  type = string
  default = "10m"
}

variable "max_agent_starting_period" {
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

variable "scheduler_type" {
  type = string
  default = "fair_share"
}

variable "preemption_enabled" {
  type = bool
  default = false
}

variable "labels" {
  type = map(string)
  default = {}
}

/******************************************
	Determined
 *****************************************/

variable "disk_size" {
  type = number
  default = 200
}

variable "disk_type" {
  type = string
  default = "pd-standard"
}

variable "environment_image" {
  type = string
}

variable "image_repo_prefix" {
  type = string
  default = "determinedai"
}

variable "det_version" {
  type = string
}

variable "det_version_key" {
  type = string
}

variable "cpu_env_image" {
  type = string
  default = ""
}

variable "gpu_env_image" {
  type = string
  default = ""
}

/******************************************
	Master
 *****************************************/

variable "scheme" {
  type = string
  default = "http"
}

variable "port" {
  type = number
  default = 8080
}

data "local_file" "master_config_template_default" {
    filename = "${path.module}/master.yaml.tmpl"
}

variable "master_config_template" {
  type = string
  default = ""
}

variable "initial_user_password" {
  type = string
  default = ""
}

/******************************************
	Database
 *****************************************/

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

variable "db_ssl_enabled" {
  type = bool
  default = true
}
