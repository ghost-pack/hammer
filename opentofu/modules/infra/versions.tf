# modules/infra/versions.tf (or at the top of main.tf)
terraform {
  required_version = ">= 1.6.0"

  required_providers {
    local = {
      source  = "hashicorp/local"
      version = "~> 2.5.0" # Use the version you actually want
    }
  }
}