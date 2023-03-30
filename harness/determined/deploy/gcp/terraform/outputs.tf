output "A1--Network" {
  value = "                           ${module.network.network_name}"
}

output "A2--Region" {
  value = "                            ${var.region}"
}

output "A3--Zone" {
  value = "                              ${module.compute.master_zone}\n"
}

output "B1--Master-Instance-Name" {
  value = "              ${module.compute.master_instance_name}"
}

output "B2--Master-Instance-Type" {
  value = "              ${var.master_instance_type}"
}

output "B3--CPU-Agent-Instance-Type" {
  value = "           ${var.aux_agent_instance_type}"
}

output "B3--GPU-Agent-Instance-Type" {
  value = "           ${var.compute_agent_instance_type}"
}

output "B4--Min-Number-of-Dynamic-Agents" {
  value = "      ${var.min_dynamic_agents}"
}

output "B4--Max-Number-of-Dynamic-Agents" {
  value = "      ${var.max_dynamic_agents}"
}

output "B5--GPUs-per-Agent" {
  value = "                    ${var.gpu_num}\n"
}

output "C1--GCS-bucket" {
  value = "${var.gcs_bucket}"
}

output "C2--Filestore-address" {
  value = "${local.filestore_address}\n"
}

output "User-Labels" {
  value = var.labels
}

output "NOTE" {
  value = "> To use the Determined CLI without needing to specify the master in each command, run:\n\n  export DET_MASTER=${module.compute.web_ui}\n"
}

output "SSH-to-Master" {
  value = "> To SSH to the Determined master instance, run:\n\n  gcloud compute ssh ${module.compute.master_instance_name}\n"
}

output "Web-UI" {
  value = module.compute.web_ui
}
