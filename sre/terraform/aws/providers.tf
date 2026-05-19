terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.50"
    }
  }

  # Uncomment in production for shared state.
  # backend "s3" {
  #   bucket = "edulms-tf-state"
  #   key    = "sre/edulms.tfstate"
  #   region = "us-east-1"
  # }
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = {
      Project   = "edulms-sre"
      ManagedBy = "terraform"
      Owner     = var.owner
    }
  }
}
