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
  value = "               ${var.cpu_agent_instance_type}"
}

output "B3--GPU-Agent-Instance-Type" {
  value = "               ${var.gpu_agent_instance_type}"
}

output "B4--Min-Number-of-Dynamic-Agents" {
  value = "      ${var.min_dynamic_agents}"
}

output "B4--Max-Number-of-Dynamic-Agents" {
  value = "      ${var.max_dynamic_agents}"
}

output "B4--Number-of-Static-Agents" {
  value = "           ${var.static_agents}"
}

output "B5--GPUs-per-Agent" {
  value = "                    ${var.gpu_num}\n"
}

output "NOTE" {
  value = "> To use the Determined CLI without needing to specify the master in each command, run:\n\n  export DET_MASTER=${module.compute.web_ui}\n"
}

output "Web-UI" {
  value = "${module.compute.web_ui}"
}
