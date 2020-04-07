// Create GCS bucket

locals {
  create_gcs_bucket =  var.gcs_bucket == null ? 1 : 0
}

resource "google_storage_bucket" "checkpoint_store" {
  name = "det-checkpoints-${var.unique_id}"
  force_destroy = true

  count = local.create_gcs_bucket
}

resource "google_storage_bucket_iam_binding" "checkpoint_editor" {
  bucket = google_storage_bucket.checkpoint_store.0.name
  role = "roles/storage.admin"
  members = [
    "serviceAccount:${var.service_account_email}"
  ]

  count = local.create_gcs_bucket
}

locals {
  gcs_bucket = local.create_gcs_bucket == 1 ? google_storage_bucket.checkpoint_store.0.name : var.gcs_bucket
}
