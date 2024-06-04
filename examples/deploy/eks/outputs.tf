output "efs_id" {
  description = "EFS ID"
  value = aws_efs_file_system.shared_efs.id
}
