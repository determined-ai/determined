// Set random integer for uniqueness

// Random integer to use
resource "random_integer" "naming_int" {
  min = 100000
  max = 999999
}

// Create GCS bucket

resource "google_storage_bucket" "checkpoint_store" {
  name = "det-checkpoints-${var.unique_id}-${random_integer.naming_int.result}"
  force_destroy = true

}

resource "google_storage_bucket_iam_binding" "checkpoint_editor" {
  bucket = google_storage_bucket.checkpoint_store.name
  role = "roles/storage.admin"
  members = [
    "serviceAccount:${var.service_account_email}"
  ]

}

locals {
  gcs_bucket = google_storage_bucket.checkpoint_store.name
}
