resource "local_file" "literature" {
  filename = var.first_file_name
  content  = var.first_file_content
}

resource "local_file" "another_resource" {
  filename = var.second_file_name
  content  = var.second_file_content
}