output "web_ui" {
  value = "${var.scheme}://${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}:${var.port}"
}

output "internal_ip" {
  value = google_compute_instance.default.network_interface.0.network_ip
}

output "master_instance_name" {
  value = google_compute_instance.default.name
}

output "master_zone" {
  value = google_compute_instance.default.zone
}
