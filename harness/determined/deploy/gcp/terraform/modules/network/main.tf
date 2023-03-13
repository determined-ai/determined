// Create Network if it doesn't exist

resource "google_compute_network" "private_network" {
  provider = google-beta

  name = "${var.network}-${var.unique_id}"
}

locals {
  network_self_link = google_compute_network.private_network.self_link
  network_name = google_compute_network.private_network.name
  subnetwork_name = var.subnetwork != null ? var.subnetwork : local.network_name
}

resource "google_compute_global_address" "private_ip_address" {
  provider = google-beta

  name          = "det-private-ip-address-${var.unique_id}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = local.network_self_link
  labels        = var.labels
}

resource "google_service_networking_connection" "private_vpc_connection" {
  provider = google-beta

  network                 = local.network_self_link
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
}

locals {
  service_networking_connection = google_service_networking_connection.private_vpc_connection.network
}
