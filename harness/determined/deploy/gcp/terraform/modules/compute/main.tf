// Create Master instance
resource "google_compute_instance" "master_instance" {
  // The resource name must match the label_value in the dynamic agent
  // provisioner config below in order for upstream tooling (det-deploy) to
  // properly clean up dynamic agents during deprovisioning. During
  // deprovisioning it filters for agents using this same label key / value.
  // We copy the same string since a resource can't reference its own name.
  name = "det-master-${var.unique_id}-${var.det_version_key}"
  machine_type = var.master_instance_type
  zone = var.zone
  tags = [var.tag_master_port, var.tag_allow_internal, var.tag_allow_ssh]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2004-lts"
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

    cat << 'EOF' > /usr/local/determined/etc/master.yaml.tmpl
    ${var.master_config_template}
    EOF

    cat << EOF > /usr/local/determined/etc/master.yaml.context
    checkpoint_storage:
      bucket: "${var.gcs_bucket}"

    db:
      user: "${var.db_username}"
      password: "${var.db_password}"
      host: "${var.database_hostname}"
      port: 5432
      name: "${var.database_name}"
      ssl_mode: ${var.database_ssl_enabled ? "verify-ca" : "disable"}
      ssl_root_cert: ${var.database_ssl_enabled ? "/etc/determined/db_ssl_root_cert.pem" : ""}

    cpu_env_image: ${var.cpu_env_image}
    gpu_env_image: ${var.gpu_env_image}

    resource_manager:
      scheduler:
        type: "${var.scheduler_type}"
        preemption: "${var.preemption_enabled}"

    resource_pools:
      pools:
        aux_pool:
          max_aux_containers_per_agent: ${var.max_aux_containers_per_agent}
          instance_type:
            machine_type: ${var.aux_agent_instance_type}
            gpu_type: ${var.gpu_type}
            gpu_num: 0
            preemptible: ${var.preemptible}
        compute_pool:
          instance_type:
            machine_type: ${var.compute_agent_instance_type}
            gpu_type: ${var.gpu_type}
            gpu_num: ${var.gpu_num}
            preemptible: ${var.preemptible}
      gcp:
        boot_disk_source_image: projects/determined-ai/global/images/${var.environment_image}
        agent_docker_image: ${var.image_repo_prefix}/determined-agent:${var.det_version}
        master_url: ${var.scheme}://internal-ip:${var.port}
        agent_docker_network: ${var.agent_docker_network}
        max_idle_agent_period: ${var.max_idle_agent_period}
        max_agent_starting_period: ${var.max_agent_starting_period}
        type: gcp
        name_prefix: det-dynamic-agent-${var.unique_id}-${var.det_version_key}-
        label_key: managed-by
        label_value: det-master-${var.unique_id}-${var.det_version_key}
        network_interface:
          network: projects/${var.project_id}/global/networks/${var.network_name}
          subnetwork: projects/${var.project_id}/regions/${var.region}/subnetworks/${var.subnetwork_name}
          external_ip: true
        network_tags: [${var.tag_allow_internal}, ${var.tag_allow_ssh}]
        service_account:
          email: "${var.service_account_email}"
          scopes: ["https://www.googleapis.com/auth/cloud-platform"]
        min_instances: ${var.min_dynamic_agents}
        max_instances: ${var.max_dynamic_agents}
        operation_timeout_period: ${var.operation_timeout_period}
        base_config:
          minCpuPlatform: ${var.min_cpu_platform_agent}
        use_cloud_logging: true
    EOF

    if [ -n "${var.filestore_address}" ]; then
      cat << EOF >> /usr/local/determined/etc/master.yaml.context
        startup_script: |
                        apt-get -y update && apt-get -y install nfs-common
                        mkdir -p /mnt/shared_fs
                        mount ${var.filestore_address} /mnt/shared_fs
                        df -h --type=nfs

    bind_mounts:
      - host_path: /mnt/shared_fs
        container_path: /run/determined/workdir/shared_fs
    EOF
    fi

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

    cat << EOF > /usr/local/determined/etc/db_ssl_root_cert.pem
    ${var.database_ssl_root_cert}
    EOF

    docker network create ${var.master_docker_network}

    touch /usr/local/determined/etc/master.yaml
    docker run \
        --name determined-master-configurator \
        --rm \
        -v /usr/local/determined/etc/:/etc/determined/ \
        --entrypoint /bin/bash \
        ${var.image_repo_prefix}/determined-master:${var.det_version} \
        -c "/usr/bin/determined-gotmpl -i /etc/determined/master.yaml.context /etc/determined/master.yaml.tmpl > /etc/determined/master.yaml"
    test $? -eq 0 || ( echo "Failed to generate master.yaml" && exit 1 )

    docker run \
        --name determined-master \
        --network ${var.master_docker_network} \
        --restart unless-stopped \
        --log-driver=gcplogs \
        -p ${var.port}:${var.port} \
        -v /usr/local/determined/etc/:/etc/determined/ \
        ${var.image_repo_prefix}/determined-master:${var.det_version}

  EOT

  network_interface {
    network = var.network_name
    subnetwork = var.subnetwork_name
    access_config {
      nat_ip = var.static_ip
    }
  }
}

// Create configured number of static agents
resource "google_compute_instance" "agent_instance" {
  name = "det-static-agent-${var.unique_id}-${var.det_version_key}-${count.index}"
  machine_type = var.compute_agent_instance_type
  zone = var.zone
  tags = [var.tag_master_port, var.tag_allow_internal, var.tag_allow_ssh]

  boot_disk {
    initialize_params {
      image = "projects/determined-ai/global/images/${var.environment_image}"
    }
  }

  service_account {
    email = var.service_account_email
    scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }

  guest_accelerator {
    type = var.gpu_type
    count = var.gpu_num
  }

  min_cpu_platform = var.min_cpu_platform_agent

  allow_stopping_for_update = true

  // Instances w/ attached GPUs must terminate on host maintenance.  Static
  // agent provisioning (and the overall deployment) fails without this.
  scheduling {
    on_host_maintenance = "TERMINATE"
  }

  metadata_startup_script = <<-EOT
    docker network create ${var.agent_docker_network}

    docker run \
        -d \
        --name determined-agent \
        --network ${var.agent_docker_network} \
        --restart unless-stopped \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -e DET_MASTER_HOST=${google_compute_instance.master_instance.network_interface.0.network_ip} \
        -e DET_RESOURCE_POOL=compute-pool \
        ${var.image_repo_prefix}/determined-agent:${var.det_version}  run --master-port=${var.port}

  EOT

  network_interface {
    network = var.network_name
    subnetwork = var.subnetwork_name
    access_config {
    }
  }

  depends_on = [
    google_compute_instance.master_instance
  ]

  count = var.static_agents
}
