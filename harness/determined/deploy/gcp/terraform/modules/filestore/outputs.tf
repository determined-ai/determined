output "address" {
  value = "${google_filestore_instance.persistence-filestore.networks[0].ip_addresses[0]}:/${google_filestore_instance.persistence-filestore.file_shares[0].name}"
}
