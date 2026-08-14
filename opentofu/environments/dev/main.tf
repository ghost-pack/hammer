module "infra" {
  source = "../modules/infra"

  first_file_name    = "dev_art_of_war.txt"
  second_file_name   = "dev_another_art.txt"
  first_file_content = "sup it's brian on dev"
  second_file_content = "sup it's brian on dev again how's it going"
}