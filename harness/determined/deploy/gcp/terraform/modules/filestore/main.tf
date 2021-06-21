// Set random integer for uniqueness

// Random integer to use
resource "random_integer" "naming_int" {
  min = 100000
  max = 999999
}

// Create Filestore instance

resource "google_filestore_instance" "persistence-filestore" {
  name = "det-filestore-${var.unique_id}-${random_integer.naming_int.result}"
  zone = var.zone
  tier = "BASIC_HDD"
  file_shares {
    capacity_gb = 1024
    name        = "only_share"
  }
  networks {
    network = var.network_name
    modes   = ["MODE_IPV4"]
  }
}

