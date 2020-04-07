output "web_ui" {
  value = module.compute.web_ui 
}

output "internal_ip" {
  value = module.compute.internal_ip
}

output "master_instance_name" {
  value = module.compute.master_instance_name
}
