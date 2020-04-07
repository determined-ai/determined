// Required

project_id = 
region = 
identifier = 
det_version = 
environment_image = 


// Optional

creds = null                              # use environment variable
network = null                            # to be created
subnetwork = null                         # to be created
gcs_bucket = null                         # to be created
service_account_email = null              # to be created
zone = null                               # inferred from region
create_static_ip = true
create_database = true
master_machine_type = "n1-standard-16"
agent_machine_type = "n1-standard-32"
gpu_type = "nvidia-tesla-v100"
gpu_num = 8
max_instances = 8
preemptible = false
agent_docker_network = "host"
master_docker_network = "determined"
max_idle_agent_period = "5m"
scheme = "http"
port = 8080
