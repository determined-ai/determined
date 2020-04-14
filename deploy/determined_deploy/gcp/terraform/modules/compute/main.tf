// Create Master instance

resource "google_compute_instance" "default" {
  name = "det-master-${var.unique_id}-${var.det_version_key}"
  machine_type = var.master_instance_type
  zone = var.zone
  tags = [var.tag_master_port, var.tag_allow_internal, var.tag_allow_ssh]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-1604-lts"
    }
  }

  service_account {
    email = var.service_account_email
    scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }

  min_cpu_platform = var.min_cpu_platform_master

  allow_stopping_for_update = true

  metadata_startup_script = <<-EOT
    mkdir -p /usr/local/determined/etc
    cat << EOF > /usr/local/determined/etc/master.yaml

    db:
      user: "${var.db_username}"
      password: "${var.db_password}"
      host: "${var.database_hostname}"
      port: 5432
      name: "${var.database_name}"

    hasura:
      secret: "${var.hasura_secret}"
      address: determined-graphql:${var.port}

    checkpoint_storage:
      type: gcs
      bucket: "${var.gcs_bucket}"

    provisioner:
      boot_disk_source_image: projects/determined-ai/global/images/${var.environment_image}
      agent_docker_image: determinedai/determined-agent:${var.det_version}
      master_url: ${var.scheme}://internal-ip:${var.port}
      agent_docker_network: ${var.agent_docker_network}
      max_idle_agent_period: ${var.max_idle_agent_period}
      provider: gcp
      name_prefix: det-agent-${var.unique_id}-
      network_interface:
        network: projects/${var.project_id}/global/networks/${var.network_name}
        subnetwork: projects/${var.project_id}/regions/${var.region}/subnetworks/${var.subnetwork_name}
        external_ip: true
      network_tags: [${var.tag_allow_internal}, ${var.tag_allow_ssh}]
      service_account:
        email: "${var.service_account_email}"
        scopes: ["https://www.googleapis.com/auth/cloud-platform"]
      instance_type:
        machine_type: ${var.agent_instance_type}
        gpu_type: ${var.gpu_type}
        gpu_num: ${var.gpu_num}
        preemptible: ${var.preemptible}
      max_instances: ${var.max_instances}
      base_config:
        minCpuPlatform: ${var.min_cpu_platform_agent}
    EOF

    apt-get remove docker docker-engine docker.io containerd runc
    apt-get update
    apt-get install -y \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg-agent \
        software-properties-common
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
    add-apt-repository \
       "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
       $(lsb_release -cs) \
       stable"
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io

    docker network create ${var.master_docker_network}

    docker run \
        -d \
        --name determined-graphql \
        --network ${var.master_docker_network} \
        --restart unless-stopped \
        -e HASURA_GRAPHQL_ADMIN_SECRET="${var.hasura_secret}" \
        -e HASURA_GRAPHQL_CONSOLE_ASSETS_DIR=/srv/console-assets \
        -e HASURA_GRAPHQL_DATABASE_URL=postgres://postgres:${var.db_password}@${var.database_hostname}:5432/${var.database_name} \
        -e HASURA_GRAPHQL_ENABLED_APIS=graphql,metadata \
        -e HASURA_GRAPHQL_ENABLED_LOG_TYPES=startup \
        -e HASURA_GRAPHQL_ENABLE_CONSOLE=false \
        -e HASURA_GRAPHQL_ENABLE_TELEMETRY=false \
        hasura/graphql-engine:v1.1.0

    docker run \
        --name determined-master \
        --network ${var.master_docker_network} \
        --restart unless-stopped \
        -p ${var.port}:${var.port} \
        -v /usr/local/determined/etc/master.yaml:/etc/determined/master.yaml \
        determinedai/determined-master:${var.det_version}

  EOT

  network_interface {
    network = var.network_name
    subnetwork = var.subnetwork_name
    access_config {
      nat_ip = var.static_ip
    }
  }
}
