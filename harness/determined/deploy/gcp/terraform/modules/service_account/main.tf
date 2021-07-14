// Create Service Account
locals {
  service_account_email = google_service_account.service_account.email
  resource_member = "serviceAccount:${local.service_account_email}"
}

resource "google_service_account" "service_account" {
  account_id   = "det-${var.unique_id}"
  display_name = "DET Service Account ${var.unique_id}"
}

resource "google_project_iam_member" "project_compute" {
  project = var.project_id
  role    = "roles/compute.admin"
  member  = local.resource_member
}

resource "google_project_iam_member" "project_service" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = local.resource_member
}

resource "google_project_iam_member" "project_iam" {
  project = var.project_id
  role    = "roles/compute.imageUser"
  member  = local.resource_member
}

resource "google_project_iam_member" "project_logging" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = local.resource_member
}
