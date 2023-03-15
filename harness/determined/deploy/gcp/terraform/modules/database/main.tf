// Miscellaneous variables to ensure dependency structure
locals {
  network_exists = var.network_self_link != null ? 1 : 0
  service_networking_connection_exists = var.service_networking_connection != null ? 1 : 0
}

// Random integer to use
resource "random_integer" "naming_int" {
  min = 100000
  max = 999999
}

// Create Database

resource "google_sql_database_instance" "db_instance" {
  name   = "det-db-instance-${var.unique_id}-${random_integer.naming_int.result}"
  database_version = var.db_version
  deletion_protection = false

  depends_on = [var.network_self_link, var.service_networking_connection]

  settings {
    tier = var.db_tier
    ip_configuration {
      ipv4_enabled = false
      private_network = var.network_self_link
    }
    database_flags {
      name = "max_connections"
      value = 96
    }
  }

}

resource "google_sql_user" "db_users" {
  name     = var.db_username
  instance = google_sql_database_instance.db_instance.name
  password = var.db_password

}

resource "google_sql_database" "database" {
  name     = "det-db-${var.unique_id}"
  instance = google_sql_database_instance.db_instance.name

  depends_on = [google_sql_user.db_users]

}

