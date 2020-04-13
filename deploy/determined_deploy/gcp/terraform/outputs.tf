output "A1--Network" {
  value = module.network.network_name
}

output "A2--Region" {
  value = var.region
}

output "A3--Zone" {
  value = "${module.compute.master_zone}\n"
}

output "B1--Master-Instance-Type" {
  value = var.master_instance_type
}

output "B2--Agent-Instance-Type" {
  value = var.agent_instance_type
}

output "B3--Max-number-of-Agents" {
  value = var.max_instances
}

output "B4--GPUs-per-Agent" {
  value = "${var.gpu_num}\n"
}

output "C1--Master-Instance-Name" {
  value = module.compute.master_instance_name
}

output "C2--Web-UI" {
  value = "${module.compute.web_ui}\n" 
}

output "NOTE" {
  value = "> To use the Determined CLI without needing to specify the master in each command:\n\nexport DET_MASTER=${module.compute.web_ui}"
}


