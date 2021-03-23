// Create Static IP

resource "google_compute_address" "static_ip" {
  name = "det-static-ip-${var.unique_id}"
  count = var.create_static_ip == true ? 1 : 0
}

