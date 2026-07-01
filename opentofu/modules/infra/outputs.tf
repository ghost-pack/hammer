output "first_file_path" {
  description = "Absolute path of the first generated file"
  value       = abspath(local_file.literature.filename)
}

output "second_file_path" {
  description = "Absolute path of the second generated file"
  value       = abspath(local_file.another_resource.filename)
}