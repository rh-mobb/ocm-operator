#
# trust policy
#
data "aws_iam_policy_document" "ocm_operator_trust_policy" {
  statement {
    sid     = "OCMOperatorTrustPolicy"
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.cluster.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "${replace(data.aws_iam_openid_connect_provider.cluster.url, "https://", "")}:sub"
      values   = ["system:serviceaccount:${var.ocm_operator_namespace}:ocm-operator-controller-manager"]
    }
  }
}

#
# operator policy
#
locals {
  ocm_policy_resource_filter = var.ocm_operator_iam_cluster_filter_prefix == null ? "*" : "arn:${data.aws_partition.current.id}:iam::${data.aws_caller_identity.current.account_id}:policy/${var.ocm_operator_iam_cluster_filter_prefix}-*"
  ocm_role_resource_filter   = var.ocm_operator_iam_cluster_filter_prefix == null ? "*" : "arn:${data.aws_partition.current.id}:iam::${data.aws_caller_identity.current.account_id}:role/${var.ocm_operator_iam_cluster_filter_prefix}-*"
}

data "aws_iam_policy_document" "ocm_operator_policy" {
  statement {
    effect    = "Allow"
    resources = ["*"]
    actions = [
      "ec2:DescribeSubnets",
      "ec2:DescribeVpcs",
      "ec2:DescribeAvailabilityZones",
      "iam:CreateOpenIDConnectProvider",
      "iam:TagOpenIDConnectProvider",
      "iam:DeleteOpenIDConnectProvider",
      "iam:ListRoles",
      "iam:ListPolicies",
      "iam:ListAttachedRolePolicies",
      "iam:ListPolicyVersions"
    ]
  }

  statement {
    effect    = "Allow"
    resources = [local.ocm_policy_resource_filter]
    actions = [
      "iam:GetPolicy",
      "iam:CreatePolicy",
      "iam:TagPolicy",
      "iam:ListPolicyTags",
      "iam:DeletePolicy"
    ]
  }

  statement {
    effect    = "Allow"
    resources = [local.ocm_role_resource_filter]
    actions = [
      "iam:GetRole",
      "iam:CreateRole",
      "iam:TagRole",
      "iam:ListRoleTags",
      "iam:DeleteRole",
    ]
  }

  statement {
    effect    = "Allow"
    resources = [local.ocm_role_resource_filter]
    actions = [
      "iam:AttachRolePolicy",
      "iam:DetachRolePolicy"
    ]

    condition {
      test     = "StringLike"
      variable = "iam:PolicyArn"
      values   = [local.ocm_policy_resource_filter]
    }
  }
}
