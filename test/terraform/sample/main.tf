variable "oidc_provider_url" {
  type = string
}

variable "cluster_name" {
  type = string
}

module "ocm_operator_iam" {
  source = "../"

  oidc_provider_url       = var.oidc_provider_url
  ocm_operator_iam_prefix = var.cluster_name
}

output "ocm_operator_iam" {
  value = module.ocm_operator_iam
}
