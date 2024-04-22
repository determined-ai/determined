
output "project" {
  value = var.project
}

output "zone" {
  value = var.zone
}

output "instance_name" {
  value = google_compute_instance.vm_instance.name
}

locals {
  external_ip = google_compute_instance.vm_instance.network_interface[0].access_config[0].nat_ip
}

output "internal_ip" {
  value = google_compute_instance.vm_instance.network_interface[0].network_ip
}

output "external_ip" {
  value = local.external_ip
}
