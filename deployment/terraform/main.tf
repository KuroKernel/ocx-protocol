terraform {
  required_providers {
    ocx = {
      source  = "ocx-protocol/ocx"
      version = "~> 1.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "ocx" {
  server_url = var.ocx_server_url
  api_key    = var.ocx_api_key
}

resource "aws_s3_bucket" "deployment_artifacts" {
  bucket = "ocx-deployment-artifacts-${random_id.bucket_suffix.hex}"
}

resource "random_id" "bucket_suffix" {
  byte_length = 8
}

resource "ocx_provenance" "infrastructure_deployment" {
  trigger_hash = sha256(jsonencode({
    bucket_name = aws_s3_bucket.deployment_artifacts.bucket
    deployment_time = timestamp()
    terraform_version = var.terraform_version
  }))
  
  ocx_server   = var.ocx_server_url
  workspace    = terraform.workspace
  storage_url  = "s3://${aws_s3_bucket.deployment_artifacts.bucket}/provenance"

  depends_on = [aws_s3_bucket.deployment_artifacts]
}

output "provenance_receipt_hash" {
  value = ocx_provenance.infrastructure_deployment.receipt_hash
}

output "deployment_storage_url" {
  value = ocx_provenance.infrastructure_deployment.storage_url
}
