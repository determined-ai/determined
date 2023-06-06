terraform {
  backend "gcs" {}

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.51.0"
    }
  }
}

provider "google" {
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
  name = var.name
  tags = [var.name, "dev"]
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

  metadata_startup_script = file("${path.module}/scripts/startup-script.sh")
}
