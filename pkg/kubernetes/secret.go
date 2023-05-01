package kubernetes

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetSecretData(ctx context.Context, c Client, name, namespace, key string) (string, error) {
	secret := &corev1.Secret{}

	if err := c.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, secret); err != nil {
		return "", fmt.Errorf(
			"unable to retrieve secret [%s/%s] from cluster - %w",
			namespace,
			name,
			err,
		)
	}

	if secret.Data == nil || len(secret.Data[key]) == 0 {
		return "", nil
	}

	return string(secret.Data[key]), nil
}
