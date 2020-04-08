data "google_compute_network" "private_network" {
  provider = google-beta

  name = var.network
}

locals {
  create_network = data.google_compute_network.private_network.name == null ? 1 : 0
}

// Create Network if it doesn't exist

resource "google_compute_network" "private_network" {
  provider = google-beta

  name = var.network
  count = local.create_network
}

locals {
  network_self_link = local.create_network == 1 ? google_compute_network.private_network.0.self_link : data.google_compute_network.private_network.self_link
  network_name = local.create_network == 1 ? google_compute_network.private_network.0.name : data.google_compute_network.private_network.name
  subnetwork_name = var.subnetwork != null ? var.subnetwork : local.network_name
}

resource "google_compute_global_address" "private_ip_address" {
  provider = google-beta

  name          = "det-private-ip-address-${var.unique_id}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = local.network_self_link
  count = local.create_network
}

resource "google_service_networking_connection" "private_vpc_connection" {
  provider = google-beta

  network                 = local.network_self_link
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.0.name]
  count = local.create_network
}

locals {
  service_networking_connection = local.create_network == 1 ? google_service_networking_connection.private_vpc_connection.0.network : null
}
