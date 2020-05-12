output "database_hostname" {
  value = google_sql_database_instance.db_instance.private_ip_address
}

output "database_name" {
  value = google_sql_database.database.name
}

output "database_ssl_root_cert" {
  value = google_sql_database_instance.db_instance.server_ca_cert.0.cert
}
