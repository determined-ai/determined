// Create Firewall rules for master port, ssh, and internal

locals {
  tags = {
    open_master_port = "det-port-${var.port}-${var.network_name}"
    allow_internal = "det-internal-${var.network_name}"
    allow_ssh = "det-ssh-${var.network_name}"
  }
}

resource "google_compute_firewall" "default_master" {
  name = local.tags.open_master_port
  network = var.network_name

  allow {
    protocol = "tcp"
    ports = [var.port]
  }

  target_tags = [local.tags.open_master_port]

}

resource "google_compute_firewall" "default_internal" {
  name = local.tags.allow_internal
  network = var.network_name
  allow {
    protocol = "tcp"
    ports = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports = ["0-65535"]
  }

  target_tags = [local.tags.allow_internal]

  source_tags = [local.tags.allow_internal]

}

resource "google_compute_firewall" "default_ssh" {
  name = local.tags.allow_ssh
  network = var.network_name
  allow {
    protocol = "tcp"
    ports = ["22"]
  }

  target_tags = [local.tags.allow_ssh]

}
