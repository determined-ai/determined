// Configure GCP provider
provider "google" {
  credentials = var.keypath != null ? file(var.keypath) : null
  project = var.project_id
  region = var.region
  zone = var.zone != null ? var.zone : "${var.region}-b"
  version = "~> 3.44.0"
}

provider "google-beta" {
  credentials = var.keypath != null ? file(var.keypath) : null
  project = var.project_id
  region = var.region
  zone = var.zone != null ? var.zone : "${var.region}-b"
  version = "~> 3.44.0"
}

locals {
  unique_id = "${var.cluster_id}"
  det_version_key = "${var.det_version_key}"
}

terraform {
  backend "local" {}
}


/******************************************
	VPC configuration
 *****************************************/

module "network" {
  source = "./modules/network"

  project_id = var.project_id
  unique_id = local.unique_id
  network = var.network
}


/******************************************
	Service Account configuration
 *****************************************/

module "service_account" {
  source = "./modules/service_account"

  project_id = var.project_id
  unique_id = local.unique_id
  service_account_email = var.service_account_email
}


/******************************************
	Static IP configuration
 *****************************************/

module "ip" {
  source = "./modules/ip"

  unique_id = local.unique_id
  create_static_ip = var.create_static_ip
}


/******************************************
	GCS configuration
 *****************************************/

module "gcs" {
  source = "./modules/gcs"

  unique_id = local.unique_id
  gcs_bucket = var.gcs_bucket
  service_account_email = module.service_account.service_account_email
}


/******************************************
	Database configuration
 *****************************************/

module "database" {
  source = "./modules/database"

  unique_id = local.unique_id
  db_tier = var.db_tier
  db_username = var.db_username
  db_password = var.db_password
  db_version = var.db_version
  network_self_link = module.network.network_self_link
  service_networking_connection = module.network.service_networking_connection
}


/******************************************
        Firewall configuration
 *****************************************/

module "firewall" {
  source = "./modules/firewall"

  unique_id = local.unique_id
  network_name = module.network.network_name
  port = var.port
}


/******************************************
	Compute configuration
 *****************************************/

module "compute" {
  source = "./modules/compute"

  unique_id = local.unique_id
  det_version_key = local.det_version_key
  project_id = var.project_id
  region = var.region
  zone = var.zone
  environment_image = var.environment_image
  det_version = var.det_version
  scheme = var.scheme
  port = var.port
  master_docker_network = var.master_docker_network
  master_instance_type = var.master_instance_type
  agent_docker_network = var.agent_docker_network
  agent_instance_type = var.agent_instance_type
  max_idle_agent_period = var.max_idle_agent_period
  max_agent_starting_period = var.max_agent_starting_period
  gpu_type = var.gpu_type
  gpu_num = var.gpu_num
  min_dynamic_agents = var.min_dynamic_agents
  max_dynamic_agents = var.max_dynamic_agents
  static_agents = var.static_agents
  min_cpu_platform_master = var.min_cpu_platform_master
  min_cpu_platform_agent = var.min_cpu_platform_agent
  preemptible = var.preemptible
  operation_timeout_period = var.operation_timeout_period
  db_username = var.db_username
  db_password = var.db_password
  scheduler_type = var.scheduler_type
  preemption_enabled = var.preemption_enabled
  cpu_env_image = var.cpu_env_image
  gpu_env_image = var.gpu_env_image

  network_name = module.network.network_name
  subnetwork_name = module.network.subnetwork_name
  static_ip = module.ip.static_ip_address
  service_account_email = module.service_account.service_account_email
  gcs_bucket = module.gcs.gcs_bucket
  database_hostname = module.database.database_hostname
  database_name = module.database.database_name
  database_ssl_enabled = var.db_ssl_enabled
  database_ssl_root_cert = module.database.database_ssl_root_cert
  tag_master_port = module.firewall.tag_master_port
  tag_allow_internal = module.firewall.tag_allow_internal
  tag_allow_ssh = module.firewall.tag_allow_ssh
}
