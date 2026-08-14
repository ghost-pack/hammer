module "infra" {
  source = "../../modules/infra"

  first_file_name    = "prod_art_of_war.txt"
  second_file_name   = "prod_another_art.txt"
  first_file_content = "sup it's brian on prod"
  second_file_content = "sup it's brian on prod again how's it going"
}