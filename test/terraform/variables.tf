variable "ocm_operator_iam_prefix" {
  type        = string
  description = "Prefix to be used for OCM operator policies, used by the operator."
}

variable "ocm_operator_iam_cluster_filter_prefix" {
  type        = string
  description = "If specified, clusters managed by the OCM operator must be created with a specific 'operatorRolesPrefix'.  If unspecified, OCM operator is allowed unbounded access to all IAM resources within the context of the permissions boundary."
  default     = null
}

variable "ocm_operator_namespace" {
  type        = string
  description = "Namespace where OCM operator is to be deployed."
  default     = "ocm-operator"
}

variable "oidc_provider_url" {
  type        = string
  description = "AWS OIDC Provider URL for the cluster where the OCM operator is running at."
}
