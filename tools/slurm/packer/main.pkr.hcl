packer {
  required_plugins {
    googlecompute = {
      version = ">= 1.0.0"
      source  = "github.com/hashicorp/googlecompute"
    }
  }
}

variables {
  ssh_username = "packer2"
}

locals {
  static_source_path = "static"
  static_dest_path   = "/tmp/static"
  reg_conf_dir      = "/etc/containers" 
  det_conf_dir       = "/etc/determined"
  slurm_sysconfdir   = "/usr/local/etc/slurm"
  launcher_job_root  = "/var/tmp/launcher"

  launcher_deb_name      = "hpe-hpc-launcher_3.3.0-0_amd64.deb"
  launcher_deb_dest_path = "${local.static_dest_path}/${local.launcher_deb_name}"

  slurm_conf_name      = "slurm.conf"
  slurm_conf_tmp_path  = "${local.static_dest_path}/${local.slurm_conf_name}"
  slurm_conf_dest_path = "${local.slurm_sysconfdir}/${local.slurm_conf_name}"

  slurm_cgroup_conf_name      = "cgroup.conf"
  slurm_cgroup_conf_tmp_path  = "${local.static_dest_path}/${local.slurm_cgroup_conf_name}"
  slurm_cgroup_conf_dest_path = "${local.slurm_sysconfdir}/${local.slurm_cgroup_conf_name}"

  det_master_conf_name      = "master.yaml"
  det_master_conf_tmp_path  = "${local.static_dest_path}/${local.det_master_conf_name}"
  det_master_conf_dest_path = "${local.det_conf_dir}/${local.det_master_conf_name}"
  
  container_registries_name      = "registries.conf"
  container_registries_tmp_path  = "${local.static_dest_path}/${local.container_registries_name}"
  container_registries_dest_path = "${local.reg_conf_dir}/${local.container_registries_name}"
}

source "googlecompute" "determined-hpc-image" {
  project_id              = "determined-ai"
  source_image_project_id = ["schedmd-slurm-public"]
  source_image_family     = "schedmd-v5-slurm-22-05-8-ubuntu-2204-lts"

  image_family      = "det-environments-slurm-ci"
  image_name        = "det-environments-slurm-ci-{{timestamp}}"
  image_description = "det environments with hpc tools to test hpc deployments"

  machine_type = "n1-standard-1"
  disk_size    = "100"
  // us-central1-c seems to be much faster/more reliable. had intermittent failures in us-west1-b
  // with IAP Tunnels being slow to come up.
  zone             = "us-central1-c"
  subnetwork       = "default"
  metadata         = { "block-project-ssh-keys" : "true" }
  omit_external_ip = true
  use_internal_ip  = true
  use_iap          = true
  // ssh_username cannot be 'packer' due to issues with nested packer builds (schedmd-slurm-public
  // images are all built with packer), ssh_clear_authorized_keys and how GCP metadata based
  // ssh-keys are provisioned.
  ssh_username              = var.ssh_username
  temporary_key_pair_type   = "ed25519"
  ssh_clear_authorized_keys = true
}

build {
  name    = "determined-hpc-image"
  sources = ["sources.googlecompute.determined-hpc-image"]

  provisioner "file" {
    source      = local.static_source_path
    destination = local.static_dest_path
  }

  provisioner "shell" {
    inline = [
      "sudo mv ${local.slurm_conf_tmp_path} ${local.slurm_conf_dest_path}",
      "sudo mv ${local.slurm_cgroup_conf_tmp_path} ${local.slurm_cgroup_conf_dest_path}",
      "sudo mkdir -p ${local.det_conf_dir}",
      "sudo mv ${local.det_master_conf_tmp_path} ${local.det_master_conf_dest_path}",
      "sudo mkdir -p ${local.launcher_job_root}",
      "sudo mkdir -p ${local.reg_conf_dir}",
      "sudo mv -f ${local.container_registries_tmp_path} ${local.container_registries_dest_path}"  
    ]
  }

  provisioner "shell" {
    script = "scripts/install-ansible.sh"
  }

  provisioner "ansible-local" {
    playbook_file   = "ansible-playbook.yml"
    extra_arguments = ["--verbose", "--extra-vars \"launcher_deb=${local.launcher_deb_dest_path}\""]
  }
}
