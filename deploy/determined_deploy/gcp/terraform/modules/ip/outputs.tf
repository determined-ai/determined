output "static_ip_address" {
  value = var.create_static_ip == true ? google_compute_address.static_ip.0.address : null
}
