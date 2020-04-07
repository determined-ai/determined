output "database_hostname" {
  value = google_sql_database_instance.db_instance.private_ip_address
}

output "database_name" {
  value = google_sql_database.database.name
}
