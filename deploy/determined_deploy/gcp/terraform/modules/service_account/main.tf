// Check if user supplied service account email (existing service account)
locals {
  create_service_account = var.service_account_email == null ? 1 : 0
}

// Create Service Account
resource "google_service_account" "service_account" {
  account_id   = "det-${var.unique_id}"
  display_name = "DET Service Account ${var.unique_id}"
  count = local.create_service_account
}

resource "google_project_iam_member" "project_compute" {
  project = var.project_id
  role    = "roles/compute.admin"
  member  = "serviceAccount:${google_service_account.service_account.0.email}"
  count = local.create_service_account
}

resource "google_project_iam_member" "project_service" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.service_account.0.email}"
  count = local.create_service_account
}

resource "google_project_iam_member" "project_iam" {
  project = var.project_id
  role    = "roles/compute.imageUser" 
  member  = "serviceAccount:${google_service_account.service_account.0.email}"
  count = local.create_service_account
}

locals {
  service_account_email = local.create_service_account == 1 ? google_service_account.service_account.0.email : var.service_account_email
}
