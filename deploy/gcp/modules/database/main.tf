// Create Database

resource "google_sql_database_instance" "db_instance" {
  name   = "det-db-instance-${var.unique_id}"
  database_version = var.db_version

  depends_on = [var.network_self_link]

  settings {
    tier = var.db_tier
    ip_configuration {
      ipv4_enabled = false
      private_network = var.network_self_link
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

