#
# provider details
#
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.20.0"
    }
  }
}

#
# account and oidc provider retrieval
#
data "aws_partition" "current" {}

data "aws_caller_identity" "current" {}

data "aws_iam_openid_connect_provider" "cluster" {
  url = var.oidc_provider_url
}

#
# iam policies and roles 
#
locals {
  ocm_operator_suffix = "OCMOperator"
}

resource "aws_iam_policy" "ocm_operator" {
  name   = "${var.ocm_operator_iam_prefix}-${local.ocm_operator_suffix}"
  policy = data.aws_iam_policy_document.ocm_operator_policy.json
}

resource "aws_iam_role" "ocm_operator" {
  name               = "${var.ocm_operator_iam_prefix}-${local.ocm_operator_suffix}"
  assume_role_policy = data.aws_iam_policy_document.ocm_operator_trust_policy.json

  lifecycle {
    ignore_changes = [permissions_boundary]
  }
}

resource "aws_iam_role_policy_attachment" "ocm_operator" {
  role       = aws_iam_role.ocm_operator.name
  policy_arn = aws_iam_policy.ocm_operator.arn
}

#
# outputs
#
output "ocm_operator_role_arn" {
  description = "This ARN is used by the OCM operator to assume and is created as part of the AWS credentials configuration."
  value       = aws_iam_role.ocm_operator.arn
}

output "ocm_operator_required_roles_prefix" {
  description = "This prefix is required in the 'spec.iam.operatorRolePrefix' field for each ROSACluster resource created by the operator.  It is used to scope down IAM permissions for the operator."
  value       = var.ocm_operator_iam_cluster_filter_prefix
}
