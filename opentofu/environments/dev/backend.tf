# environments/dev/backend.tf
terraform {
  # Pin the engine version. OpenTofu 1.6+ is compatible with Terraform 1.6 syntax.
  required_version = ">= 1.6.0"

  backend "gcs" {
    # overridden in CI/CD
  }
}