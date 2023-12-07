terraform {
  backend "gcs" {}

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.51.0"
    }
  }
}

locals {
    gpu_enabled            = var.gpus != null
}

provider "google" {
  project = var.project
  region  = var.region
  zone    = var.zone
}

provider "google-beta" {
  project = var.project
  region  = var.region
  zone    = var.zone
}

resource "google_compute_network" "vpc_network" {
  name = var.name
}

// TODO: Would rather use IAP, but I don't have permissions to mess with it.
resource "google_compute_firewall" "ssh-rule" {
  name          = var.name
  network       = google_compute_network.vpc_network.name
  target_tags   = [var.name]
  source_ranges = [var.ssh_allow_ip]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  depends_on = [
    google_compute_network.vpc_network
  ]
}

resource "google_compute_instance" "vm_instance" {
  name     = var.name
  provider = google-beta
  tags     = [var.name, "dev"]
  metadata = {
    ssh-keys = "${var.ssh_user}:${file(var.ssh_key_pub)}"
  }

  machine_type = var.machine_type

  boot_disk {
    initialize_params {
      image = var.boot_disk
    }
  }

  network_interface {
    network = google_compute_network.vpc_network.name
    access_config {
    }
  }

  allow_stopping_for_update = var.allow_stopping_for_update

  dynamic "guest_accelerator" {
    for_each = local.gpu_enabled ? [var.gpus] : []
    content {
      type  = guest_accelerator.value.type
      count = guest_accelerator.value.count
    }
  }


  scheduling {
    max_run_duration {
      // Gives two hours (by default) of runtime before the box closes
      // Useful for CircleCI tests in case the job is cancelled
      seconds = var.vmLifetimeSeconds
    }
    instance_termination_action = "DELETE"
    on_host_maintenance = "TERMINATE"
  }

  

  metadata_startup_script = templatefile("${path.module}/scripts/startup-script.sh", { WORKLOAD_MANAGER = var.workload_manager })
}
